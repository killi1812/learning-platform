package quiz

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrQuizNotFound     = errors.New("quiz not found")
	ErrModuleNotFound   = errors.New("module not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrQuestionNotFound = errors.New("question not found")
	ErrNoActiveAttempt  = errors.New("no attempt in progress")
	ErrQuizClosed       = errors.New("quiz is not open right now")
	ErrNoAttemptsLeft   = errors.New("no attempts left")
	ErrInvalidQuestion  = errors.New("invalid question")
	ErrInvalidAnswer    = errors.New("invalid answer")
	ErrInvalidQuiz      = errors.New("invalid quiz settings")
	ErrNoQuestions      = errors.New("quiz needs at least one question to publish")
)

const (
	trueFalseOptionCount = 2
	minChoiceOptions     = 2
	maxChoiceOptions     = 8
)

type IQuizService interface {
	// Teacher
	CreateQuiz(courseUuid, moduleUuid uuid.UUID, title string) (*EditQuizDto, error)
	GetEditQuiz(courseUuid, contentUuid uuid.UUID) (*EditQuizDto, error)
	UpdateQuiz(courseUuid, contentUuid uuid.UUID, req UpdateQuizRequest) (*EditQuizDto, error)
	ReplaceQuestions(courseUuid, contentUuid uuid.UUID, req ReplaceQuestionsRequest) (*EditQuizDto, error)
	Publish(courseUuid, contentUuid uuid.UUID) (*EditQuizDto, error)
	// NotifyDueQuizzes notifies students of published quizzes that are open now and not yet announced.
	NotifyDueQuizzes() error
	DeleteQuiz(courseUuid, contentUuid uuid.UUID) error
	Results(courseUuid, contentUuid uuid.UUID) ([]StudentResultRowDto, error)
	// Everyone enrolled
	ListAssignments(courseUuid, userUuid uuid.UUID, isTeacher bool) ([]AssignmentDto, error)
	// Student
	StartAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*AttemptDto, error)
	GetAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*AttemptDto, error)
	SaveAnswer(courseUuid, contentUuid, userUuid, questionUuid uuid.UUID, selected []int) error
	SubmitAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*ResultDto, error)
}

type QuizService struct {
	db       *gorm.DB
	notifier Notifier
	logger   *zap.SugaredLogger
}

func NewQuizService(db *gorm.DB, notifier Notifier, logger *zap.SugaredLogger) IQuizService {
	return &QuizService{db: db, notifier: notifier, logger: logger}
}

// ---------- loading helpers ----------

// loadQuiz finds a quiz content item that belongs to the given course.
func (s *QuizService) loadQuiz(courseUuid, contentUuid uuid.UUID) (*content.Content, error) {
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("contents.uuid = ? AND courses.uuid = ? AND contents.type = ?", contentUuid, courseUuid, content.TypeAssignment).
		Preload("Module").
		Preload("Assignment.Quiz.Questions", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC, id ASC")
		}).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuizNotFound
		}
		return nil, err
	}
	if cnt.Assignment == nil || cnt.Assignment.Type != content.AssignmentQuiz || cnt.Assignment.Quiz == nil {
		return nil, ErrQuizNotFound
	}
	return &cnt, nil
}

// loadOpenQuiz is loadQuiz for students: the quiz must be published and inside its window.
func (s *QuizService) loadOpenQuiz(courseUuid, contentUuid uuid.UUID) (*content.Content, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	if !cnt.Assignment.Quiz.Published {
		return nil, ErrQuizNotFound
	}
	now := time.Now()
	if cnt.VisibleFrom != nil && now.Before(*cnt.VisibleFrom) {
		return nil, ErrQuizClosed
	}
	if cnt.VisibleUntil != nil && now.After(*cnt.VisibleUntil) {
		return nil, ErrQuizClosed
	}
	return cnt, nil
}

func (s *QuizService) findUser(userUuid uuid.UUID) (*user.User, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func toEditDto(cnt *content.Content) *EditQuizDto {
	quiz := cnt.Assignment.Quiz
	dto := &EditQuizDto{
		Uuid:         cnt.Uuid.String(),
		Title:        cnt.Title,
		Published:    quiz.Published,
		VisibleFrom:  cnt.VisibleFrom,
		VisibleUntil: cnt.VisibleUntil,
		PassPercent:  quiz.PassPercent,
		MaxAttempts:  quiz.MaxAttempts,
		Questions:    make([]EditQuestionDto, 0, len(quiz.Questions)),
	}
	if cnt.Module != nil {
		dto.ModuleUuid = cnt.Module.Uuid.String()
	}
	for _, q := range quiz.Questions {
		dto.Questions = append(dto.Questions, toEditQuestion(q))
	}
	return dto
}

// ---------- teacher: create and edit ----------

func (s *QuizService) CreateQuiz(courseUuid, moduleUuid uuid.UUID, title string) (*EditQuizDto, error) {
	var m course.Module
	err := s.db.
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("modules.uuid = ? AND courses.uuid = ?", moduleUuid, courseUuid).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}

	cnt := &content.Content{ModuleID: m.ID, Title: strings.TrimSpace(title), Type: content.TypeAssignment}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var maxPos *int
		if err := tx.Model(&content.Content{}).Where("module_id = ?", m.ID).Select("MAX(position)").Scan(&maxPos).Error; err != nil {
			return err
		}
		cnt.Position = 0
		if maxPos != nil {
			cnt.Position = *maxPos + 1
		}
		if err := tx.Create(cnt).Error; err != nil {
			return fmt.Errorf("create content: %w", err)
		}
		asgn := &content.Assignment{ContentID: cnt.ID, Type: content.AssignmentQuiz}
		if err := tx.Create(asgn).Error; err != nil {
			return fmt.Errorf("create assignment: %w", err)
		}
		return tx.Create(&content.Quiz{ContentID: cnt.ID, MaxAttempts: 1, PassPercent: 50}).Error
	})
	if err != nil {
		s.logger.Errorf("Failed to create quiz: %v", err)
		return nil, err
	}
	return s.GetEditQuiz(courseUuid, cnt.Uuid)
}

func (s *QuizService) GetEditQuiz(courseUuid, contentUuid uuid.UUID) (*EditQuizDto, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	return toEditDto(cnt), nil
}

// resetAttempts removes every attempt and answer of a quiz so students must retake it.
func resetAttempts(tx *gorm.DB, quizID uint) error {
	var ids []uint
	if err := tx.Model(&Attempt{}).Where("quiz_id = ?", quizID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Unscoped().Where("attempt_id IN ?", ids).Delete(&Answer{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("id IN ?", ids).Delete(&Attempt{}).Error
}

func (s *QuizService) UpdateQuiz(courseUuid, contentUuid uuid.UUID, req UpdateQuizRequest) (*EditQuizDto, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	if req.VisibleFrom != nil && req.VisibleUntil != nil && req.VisibleUntil.Before(*req.VisibleFrom) {
		return nil, fmt.Errorf("%w: end date is before start date", ErrInvalidQuiz)
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := resetAttempts(tx, cnt.ID); err != nil {
			return err
		}
		if err := rearmUpdateNotice(tx, cnt.ID); err != nil {
			return err
		}
		if err := tx.Model(&content.Content{}).Where("id = ?", cnt.ID).Updates(map[string]any{
			"title":         strings.TrimSpace(req.Title),
			"visible_from":  req.VisibleFrom,
			"visible_until": req.VisibleUntil,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&content.Quiz{}).Where("content_id = ?", cnt.ID).Updates(map[string]any{
			"pass_percent": req.PassPercent,
			"max_attempts": req.MaxAttempts,
		}).Error
	})
	if err != nil {
		s.logger.Errorf("Failed to update quiz: %v", err)
		return nil, err
	}
	return s.GetEditQuiz(courseUuid, contentUuid)
}

// buildQuestion validates teacher input and converts it to a stored question.
func buildQuestion(in QuestionInput, position int, quizID uint) (content.Question, error) {
	q := content.Question{
		QuizID:   quizID,
		Position: position,
		Type:     in.Type,
		Prompt:   strings.TrimSpace(in.Prompt),
		Points:   1,
	}
	if q.Prompt == "" {
		return q, fmt.Errorf("%w: empty prompt", ErrInvalidQuestion)
	}

	var options []string
	switch in.Type {
	case content.TypeTrueFalse:
		options = []string{"True", "False"}
		if len(in.Correct) != 1 {
			return q, fmt.Errorf("%w: true/false needs exactly one correct option", ErrInvalidQuestion)
		}
	case content.TypeMultipleChoice:
		for _, o := range in.Options {
			o = strings.TrimSpace(o)
			if o == "" {
				return q, fmt.Errorf("%w: empty option text", ErrInvalidQuestion)
			}
			options = append(options, o)
		}
		if len(options) < minChoiceOptions || len(options) > maxChoiceOptions {
			return q, fmt.Errorf("%w: need %d to %d options", ErrInvalidQuestion, minChoiceOptions, maxChoiceOptions)
		}
		if len(in.Correct) == 0 {
			return q, fmt.Errorf("%w: mark at least one correct option", ErrInvalidQuestion)
		}
	default:
		return q, fmt.Errorf("%w: unsupported type %q", ErrInvalidQuestion, in.Type)
	}

	if !validIndices(in.Correct, len(options)) {
		return q, fmt.Errorf("%w: correct option index out of range", ErrInvalidQuestion)
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return q, err
	}
	keyJSON, err := json.Marshal(in.Correct)
	if err != nil {
		return q, err
	}
	q.Options = string(optionsJSON)
	q.AnswerKey = string(keyJSON)
	return q, nil
}

// validIndices checks every index is inside [0,count) and none repeats.
func validIndices(indices []int, count int) bool {
	seen := make(map[int]bool, len(indices))
	for _, i := range indices {
		if i < 0 || i >= count || seen[i] {
			return false
		}
		seen[i] = true
	}
	return true
}

func (s *QuizService) ReplaceQuestions(courseUuid, contentUuid uuid.UUID, req ReplaceQuestionsRequest) (*EditQuizDto, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}

	questions := make([]content.Question, 0, len(req.Questions))
	for i, in := range req.Questions {
		q, err := buildQuestion(in, i, cnt.ID)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := resetAttempts(tx, cnt.ID); err != nil {
			return err
		}
		if err := rearmUpdateNotice(tx, cnt.ID); err != nil {
			return err
		}
		if err := tx.Unscoped().Where("quiz_id = ?", cnt.ID).Delete(&content.Question{}).Error; err != nil {
			return err
		}
		if len(questions) == 0 {
			return nil
		}
		return tx.Create(&questions).Error
	})
	if err != nil {
		s.logger.Errorf("Failed to replace questions: %v", err)
		return nil, err
	}
	return s.GetEditQuiz(courseUuid, contentUuid)
}

func (s *QuizService) Publish(courseUuid, contentUuid uuid.UUID) (*EditQuizDto, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	if len(cnt.Assignment.Quiz.Questions) == 0 {
		return nil, ErrNoQuestions
	}
	// Clearing notified_at re-arms the announcement, so a republish after an edit notifies again.
	err = s.db.Model(&content.Quiz{}).Where("content_id = ?", cnt.ID).
		Updates(map[string]any{"published": true, "notified_at": nil, "notify_update": false}).Error
	if err != nil {
		return nil, err
	}
	// Open now: students are told immediately. Future start: the worker announces it later.
	if err := s.NotifyDueQuizzes(); err != nil {
		s.logger.Errorf("Failed to notify students after publish: %v", err)
	}
	return s.GetEditQuiz(courseUuid, contentUuid)
}

func (s *QuizService) DeleteQuiz(courseUuid, contentUuid uuid.UUID) error {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := resetAttempts(tx, cnt.ID); err != nil {
			return err
		}
		if err := tx.Unscoped().Where("quiz_id = ?", cnt.ID).Delete(&content.Question{}).Error; err != nil {
			return err
		}
		if err := tx.Where("content_id = ?", cnt.ID).Delete(&content.Quiz{}).Error; err != nil {
			return err
		}
		if err := tx.Where("content_id = ?", cnt.ID).Delete(&content.Assignment{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&content.Content{}, cnt.ID).Error
	})
}

// ---------- results ----------

// bestOf returns the highest-percent attempt and the number of attempts given.
func bestOf(attempts []Attempt) (best *Attempt, count int) {
	for i := range attempts {
		count++
		if best == nil || attempts[i].Percent > best.Percent {
			best = &attempts[i]
		}
	}
	return best, count
}

func (s *QuizService) submittedAttempts(quizIDs []uint) (map[uint]map[uint][]Attempt, error) {
	byQuizThenUser := map[uint]map[uint][]Attempt{}
	if len(quizIDs) == 0 {
		return byQuizThenUser, nil
	}
	var attempts []Attempt
	if err := s.db.Where("quiz_id IN ? AND submitted_at IS NOT NULL", quizIDs).Find(&attempts).Error; err != nil {
		return nil, err
	}
	for _, a := range attempts {
		if byQuizThenUser[a.QuizID] == nil {
			byQuizThenUser[a.QuizID] = map[uint][]Attempt{}
		}
		byQuizThenUser[a.QuizID][a.UserID] = append(byQuizThenUser[a.QuizID][a.UserID], a)
	}
	return byQuizThenUser, nil
}

func (s *QuizService) Results(courseUuid, contentUuid uuid.UUID) ([]StudentResultRowDto, error) {
	cnt, err := s.loadQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}

	var enrollments []course.Enrollment
	if err := s.db.Preload("User").
		Where("course_id = ? AND role = ?", cnt.Module.CourseID, course.RoleStudent).
		Find(&enrollments).Error; err != nil {
		return nil, err
	}

	byUser, err := s.submittedAttempts([]uint{cnt.ID})
	if err != nil {
		return nil, err
	}

	rows := make([]StudentResultRowDto, 0, len(enrollments))
	for _, e := range enrollments {
		if e.User == nil {
			continue
		}
		row := StudentResultRowDto{Username: e.User.Username, Email: e.User.Email}
		best, count := bestOf(byUser[cnt.ID][e.UserID])
		row.Attempts = count
		if best != nil {
			percent, passed := best.Percent, best.Passed
			row.Percent, row.Passed = &percent, &passed
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *QuizService) ListAssignments(courseUuid, userUuid uuid.UUID, isTeacher bool) ([]AssignmentDto, error) {
	var c course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, course.ErrCourseNotFound
		}
		return nil, err
	}

	var cnts []content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Where("modules.course_id = ? AND contents.type = ?", c.ID, content.TypeAssignment).
		Preload("Module").
		Preload("Assignment.Quiz").
		Order("modules.position ASC, contents.position ASC, contents.id ASC").
		Find(&cnts).Error
	if err != nil {
		return nil, err
	}

	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}

	var quizIDs []uint
	var handinIDs []uint
	for _, cnt := range cnts {
		if cnt.Assignment != nil {
			if cnt.Assignment.Type == content.AssignmentQuiz && cnt.Assignment.Quiz != nil {
				quizIDs = append(quizIDs, cnt.ID)
			} else if cnt.Assignment.Type == content.AssignmentHandIn {
				handinIDs = append(handinIDs, cnt.ID)
			}
		}
	}
	byQuiz, err := s.submittedAttempts(quizIDs)
	if err != nil {
		return nil, err
	}

	type handinAttemptCount struct {
		ContentID uint
		Count     int
	}
	var handinCounts []handinAttemptCount
	if len(handinIDs) > 0 {
		_ = s.db.Table("individual_submissions").
			Select("content_id, count(*) as count").
			Where("content_id IN ? AND user_id = ?", handinIDs, u.ID).
			Group("content_id").
			Scan(&handinCounts).Error
	}
	handinAttemptMap := make(map[uint]int, len(handinCounts))
	for _, hc := range handinCounts {
		handinAttemptMap[hc.ContentID] = hc.Count
	}

	list := make([]AssignmentDto, 0, len(cnts))
	for _, cnt := range cnts {
		if cnt.Assignment == nil {
			continue
		}

		if cnt.Assignment.Type == content.AssignmentHandIn {
			dto := AssignmentDto{
				Uuid:                   cnt.Uuid.String(),
				Type:                   string(content.AssignmentHandIn),
				Title:                  cnt.Title,
				Published:              true,
				VisibleFrom:            cnt.VisibleFrom,
				VisibleUntil:           cnt.VisibleUntil,
				Deadline:               cnt.Assignment.Deadline,
				AcceptsLateSubmissions: cnt.Assignment.AcceptsLateSubmissions,
				AttemptsUsed:           handinAttemptMap[cnt.ID],
			}
			if cnt.Module != nil {
				dto.ModuleUuid = cnt.Module.Uuid.String()
				dto.ModuleTitle = cnt.Module.Title
			}
			list = append(list, dto)
			continue
		}

		if cnt.Assignment.Type != content.AssignmentQuiz || cnt.Assignment.Quiz == nil {
			continue
		}
		quiz := cnt.Assignment.Quiz
		if !isTeacher && !quiz.Published {
			continue
		}
		dto := AssignmentDto{
			Uuid:         cnt.Uuid.String(),
			Type:         string(content.AssignmentQuiz),
			Title:        cnt.Title,
			Published:    quiz.Published,
			VisibleFrom:  cnt.VisibleFrom,
			VisibleUntil: cnt.VisibleUntil,
			PassPercent:  quiz.PassPercent,
			MaxAttempts:  quiz.MaxAttempts,
		}
		if cnt.Module != nil {
			dto.ModuleUuid = cnt.Module.Uuid.String()
			dto.ModuleTitle = cnt.Module.Title
		}
		best, count := bestOf(byQuiz[cnt.ID][u.ID])
		dto.AttemptsUsed = count
		if best != nil {
			dto.MyResult = &ResultDto{Percent: best.Percent, Passed: best.Passed, AttemptsUsed: count, MaxAttempts: quiz.MaxAttempts}
		}
		list = append(list, dto)
	}
	return list, nil
}

// ---------- student: attempts ----------

func (s *QuizService) activeAttempt(userID, quizID uint) (*Attempt, error) {
	var a Attempt
	err := s.db.Where("user_id = ? AND quiz_id = ? AND submitted_at IS NULL", userID, quizID).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoActiveAttempt
		}
		return nil, err
	}
	return &a, nil
}

func (s *QuizService) buildAttemptDto(cnt *content.Content, a *Attempt) (*AttemptDto, error) {
	var answers []Answer
	if err := s.db.Where("attempt_id = ?", a.ID).Find(&answers).Error; err != nil {
		return nil, err
	}
	uuidByQuestion := map[uint]string{}
	questions := make([]StudentQuestionDto, 0, len(cnt.Assignment.Quiz.Questions))
	for _, q := range cnt.Assignment.Quiz.Questions {
		uuidByQuestion[q.ID] = q.Uuid.String()
		questions = append(questions, toStudentQuestion(q))
	}
	selected := map[string][]int{}
	for _, ans := range answers {
		if qUuid, ok := uuidByQuestion[ans.QuestionID]; ok {
			selected[qUuid] = decodeIndices(ans.Selected)
		}
	}
	return &AttemptDto{Questions: questions, Answers: selected, StartedAt: a.StartedAt}, nil
}

func (s *QuizService) StartAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*AttemptDto, error) {
	cnt, err := s.loadOpenQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}

	active, err := s.activeAttempt(u.ID, cnt.ID)
	if err == nil {
		return s.buildAttemptDto(cnt, active)
	}
	if !errors.Is(err, ErrNoActiveAttempt) {
		return nil, err
	}

	var used int64
	if err := s.db.Model(&Attempt{}).Where("user_id = ? AND quiz_id = ? AND submitted_at IS NOT NULL", u.ID, cnt.ID).Count(&used).Error; err != nil {
		return nil, err
	}
	if int(used) >= cnt.Assignment.Quiz.MaxAttempts {
		return nil, ErrNoAttemptsLeft
	}

	a := &Attempt{UserID: u.ID, QuizID: cnt.ID, StartedAt: time.Now()}
	if err := s.db.Create(a).Error; err != nil {
		return nil, err
	}
	return s.buildAttemptDto(cnt, a)
}

func (s *QuizService) GetAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*AttemptDto, error) {
	cnt, err := s.loadOpenQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}
	active, err := s.activeAttempt(u.ID, cnt.ID)
	if err != nil {
		return nil, err
	}
	return s.buildAttemptDto(cnt, active)
}

func (s *QuizService) SaveAnswer(courseUuid, contentUuid, userUuid, questionUuid uuid.UUID, selected []int) error {
	cnt, err := s.loadOpenQuiz(courseUuid, contentUuid)
	if err != nil {
		return err
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return err
	}
	active, err := s.activeAttempt(u.ID, cnt.ID)
	if err != nil {
		return err
	}

	var question *content.Question
	for i := range cnt.Assignment.Quiz.Questions {
		if cnt.Assignment.Quiz.Questions[i].Uuid == questionUuid {
			question = &cnt.Assignment.Quiz.Questions[i]
			break
		}
	}
	if question == nil {
		return ErrQuestionNotFound
	}

	if !validIndices(selected, len(decodeOptions(question.Options))) {
		return ErrInvalidAnswer
	}
	if question.Type == content.TypeTrueFalse && len(selected) != 1 {
		return ErrInvalidAnswer
	}

	selectedJSON, err := json.Marshal(selected)
	if err != nil {
		return err
	}

	var existing Answer
	err = s.db.Where("attempt_id = ? AND question_id = ?", active.ID, question.ID).First(&existing).Error
	switch {
	case err == nil:
		return s.db.Model(&existing).Update("selected", string(selectedJSON)).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.Create(&Answer{AttemptID: active.ID, QuestionID: question.ID, Selected: string(selectedJSON)}).Error
	default:
		return err
	}
}

// SubmitAttempt grades the in-progress attempt. Grading lives only here, on the server.
func (s *QuizService) SubmitAttempt(courseUuid, contentUuid, userUuid uuid.UUID) (*ResultDto, error) {
	cnt, err := s.loadOpenQuiz(courseUuid, contentUuid)
	if err != nil {
		return nil, err
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}
	active, err := s.activeAttempt(u.ID, cnt.ID)
	if err != nil {
		return nil, err
	}

	var answers []Answer
	if err := s.db.Where("attempt_id = ?", active.ID).Find(&answers).Error; err != nil {
		return nil, err
	}
	selectedByQuestion := make(map[uint][]int, len(answers))
	for _, a := range answers {
		selectedByQuestion[a.QuestionID] = decodeIndices(a.Selected)
	}

	quiz := cnt.Assignment.Quiz
	earned, possible := 0, 0
	for _, q := range quiz.Questions {
		// A question the student skipped has no entry: nil selection scores 0.
		selected := selectedByQuestion[q.ID]
		e, p := scoreQuestion(q.Type, len(decodeOptions(q.Options)), decodeIndices(q.AnswerKey), selected)
		earned += e
		possible += p
	}

	percent := percentOf(earned, possible)
	now := time.Now()
	err = s.db.Model(active).Updates(map[string]any{
		"submitted_at": now,
		"percent":      percent,
		"passed":       percent >= float64(quiz.PassPercent),
	}).Error
	if err != nil {
		return nil, err
	}

	var submitted []Attempt
	if err := s.db.Where("user_id = ? AND quiz_id = ? AND submitted_at IS NOT NULL", u.ID, cnt.ID).Find(&submitted).Error; err != nil {
		return nil, err
	}
	// The student sees the attempt just submitted. The best attempt still counts and is what the
	// assignment list and the teacher results show.
	return &ResultDto{
		Percent:      percent,
		Passed:       percent >= float64(quiz.PassPercent),
		AttemptsUsed: len(submitted),
		MaxAttempts:  quiz.MaxAttempts,
	}, nil
}
