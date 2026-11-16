package quiz_test

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// recordingNotifier remembers every notification instead of delivering it.
type recordingNotifier struct {
	mu   sync.Mutex
	sent map[uuid.UUID][]notification.NotificationContent
	fail map[uuid.UUID]bool
}

func (r *recordingNotifier) NotifySystem(recipient uuid.UUID, content notification.NotificationContent) (*notification.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail[recipient] {
		return nil, errors.New("delivery failed")
	}
	r.sent[recipient] = append(r.sent[recipient], content)
	return &notification.Notification{}, nil
}

func (r *recordingNotifier) NotifySystemMany(recipients []uuid.UUID, content notification.NotificationContent) ([]notification.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]notification.Notification, 0, len(recipients))
	for _, rec := range recipients {
		if r.fail[rec] {
			continue
		}
		r.sent[rec] = append(r.sent[rec], content)
		res = append(res, notification.Notification{})
	}
	return res, nil
}

func (r *recordingNotifier) count(recipient uuid.UUID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sent[recipient])
}

type fixture struct {
	db      *gorm.DB
	notes   *recordingNotifier
	svc     quiz.IQuizService
	course  *course.Course
	module  *course.Module
	teacher *user.User
	student *user.User
}

func setup(t *testing.T) *fixture {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Module{},
		&content.Content{},
		&content.Assignment{},
		&content.Quiz{},
		&content.Question{},
		&quiz.Attempt{},
		&quiz.Answer{},
	)

	c := &course.Course{Uuid: uuid.New(), Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	require.NoError(t, db.Create(c).Error)
	m := &course.Module{Uuid: uuid.New(), CourseID: c.ID, Title: "Week 1", Position: 1}
	require.NoError(t, db.Create(m).Error)

	mkUser := func(name string, role course.CourseRole) *user.User {
		u := &user.User{Uuid: uuid.New(), Username: name, Email: name + "@uib.no", PasswordHash: "x"}
		require.NoError(t, db.Create(u).Error)
		require.NoError(t, db.Create(&course.Enrollment{UserID: u.ID, CourseID: c.ID, Role: role, EnrolledAt: time.Now()}).Error)
		return u
	}

	notes := &recordingNotifier{sent: map[uuid.UUID][]notification.NotificationContent{}, fail: map[uuid.UUID]bool{}}
	return &fixture{
		db:      db,
		notes:   notes,
		svc:     quiz.NewQuizService(db, notes, zap.NewNop().Sugar()),
		course:  c,
		module:  m,
		teacher: mkUser("teacher", course.RoleTeacher),
		student: mkUser("student", course.RoleStudent),
	}
}

var animalQuestion = quiz.QuestionInput{
	Type:    content.TypeMultipleChoice,
	Prompt:  "Which animal can fly?",
	Options: []string{"tiger", "bird", "fly", "car"},
	Correct: []int{1, 2}, // 0110
}

var trueFalseQuestion = quiz.QuestionInput{
	Type:    content.TypeTrueFalse,
	Prompt:  "Go is compiled",
	Correct: []int{0}, // True
}

// publishedQuiz creates and publishes a quiz with the given questions.
func (f *fixture) publishedQuiz(t *testing.T, questions ...quiz.QuestionInput) *quiz.EditQuizDto {
	dto, err := f.svc.CreateQuiz(f.course.Uuid, f.module.Uuid, "Quiz 1")
	require.NoError(t, err)
	cu := uuid.MustParse(dto.Uuid)
	_, err = f.svc.ReplaceQuestions(f.course.Uuid, cu, quiz.ReplaceQuestionsRequest{Questions: questions})
	require.NoError(t, err)
	dto, err = f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)
	return dto
}

// attempt answers every question with the selections in order and submits.
func (f *fixture) attempt(t *testing.T, quizUuid string, selections ...[]int) (*quiz.ResultDto, error) {
	qu := uuid.MustParse(quizUuid)
	a, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	for i, sel := range selections {
		require.NoError(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, uuid.MustParse(a.Questions[i].Uuid), sel))
	}
	return f.svc.SubmitAttempt(f.course.Uuid, qu, f.student.Uuid)
}

func TestGrading_MultipleChoiceCountsMatchingPositions(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion)

	// key 0110, answer 0101: positions 1 and 2 match = 2 of 4 = 50%
	res, err := f.attempt(t, q.Uuid, []int{1, 3})
	require.NoError(t, err)
	assert.InDelta(t, 50.0, res.Percent, 0.001)
	assert.True(t, res.Passed, "default pass mark is 50")
}

func TestGrading_PerfectAndZero(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)
	require.NoError(t, f.db.Model(&content.Quiz{}).Where("content_id > 0").Update("max_attempts", 3).Error)

	res, err := f.attempt(t, q.Uuid, []int{1, 2}, []int{0})
	require.NoError(t, err)
	assert.InDelta(t, 100.0, res.Percent, 0.001)

	// worst answer: 1001 on the multiple choice (all four positions wrong) and False
	res, err = f.attempt(t, q.Uuid, []int{0, 3}, []int{1})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, res.Percent, 0.001, "result shows the attempt just submitted")
	assert.False(t, res.Passed)
	assert.Equal(t, 2, res.AttemptsUsed)

	// the best attempt still counts in the assignment list
	list, err := f.svc.ListAssignments(f.course.Uuid, f.student.Uuid, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].MyResult)
	assert.InDelta(t, 100.0, list[0].MyResult.Percent, 0.001, "best attempt counts")
	assert.True(t, list[0].MyResult.Passed)
}

func TestGrading_PassMarkDecidesPassed(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion)
	_, err := f.svc.UpdateQuiz(f.course.Uuid, uuid.MustParse(q.Uuid), quiz.UpdateQuizRequest{
		Title: "Quiz 1", PassPercent: 75, MaxAttempts: 1,
	})
	require.NoError(t, err)

	res, err := f.attempt(t, q.Uuid, []int{1, 3}) // 50%
	require.NoError(t, err)
	assert.False(t, res.Passed)
}

func TestAttempt_LimitAndSingleInProgress(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	qu := uuid.MustParse(q.Uuid)

	first, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	again, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	assert.True(t, first.StartedAt.Equal(again.StartedAt), "second start resumes the same attempt")

	var count int64
	f.db.Model(&quiz.Attempt{}).Count(&count)
	assert.Equal(t, int64(1), count)

	_, err = f.attempt(t, q.Uuid, []int{0})
	require.NoError(t, err)
	_, err = f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	assert.ErrorIs(t, err, quiz.ErrNoAttemptsLeft)
}

func TestSubmit_SkippedQuestionScoresZero(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)

	// first question perfect (4 of 4), second left blank (0 of 1): 4 of 5 points
	res, err := f.attempt(t, q.Uuid, []int{1, 2})
	require.NoError(t, err)
	assert.InDelta(t, 80.0, res.Percent, 0.001)
}

func TestSubmit_NothingAnsweredIsZeroPercent(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)

	res, err := f.attempt(t, q.Uuid)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, res.Percent, 0.001)
	assert.False(t, res.Passed)
}

func TestGrading_EmptySelectionOnMultipleChoiceIsZeroNotPartial(t *testing.T) {
	f := setup(t)
	// key 0010: leaving everything unmarked (0000) would match 3 of 4 positions by position rule
	q := f.publishedQuiz(t, quiz.QuestionInput{
		Type:    content.TypeMultipleChoice,
		Prompt:  "Pick the third",
		Options: []string{"a", "b", "c", "d"},
		Correct: []int{2},
	})
	qu := uuid.MustParse(q.Uuid)
	a, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	// student unticks everything: saved as an empty selection
	require.NoError(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, uuid.MustParse(a.Questions[0].Uuid), []int{}))

	res, err := f.svc.SubmitAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, res.Percent, 0.001)
}

func TestSaveAnswer_Validates(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)
	qu := uuid.MustParse(q.Uuid)
	a, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)

	mc := uuid.MustParse(a.Questions[0].Uuid)
	tf := uuid.MustParse(a.Questions[1].Uuid)
	assert.ErrorIs(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, mc, []int{9}), quiz.ErrInvalidAnswer)
	assert.ErrorIs(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, mc, []int{1, 1}), quiz.ErrInvalidAnswer)
	assert.ErrorIs(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, tf, []int{0, 1}), quiz.ErrInvalidAnswer)
	assert.ErrorIs(t, f.svc.SaveAnswer(f.course.Uuid, qu, f.student.Uuid, uuid.New(), []int{0}), quiz.ErrQuestionNotFound)
}

func TestWindow_BlocksOutsideDates(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	qu := uuid.MustParse(q.Uuid)

	setWindow := func(from, until time.Time) {
		_, err := f.svc.UpdateQuiz(f.course.Uuid, qu, quiz.UpdateQuizRequest{
			Title: "Quiz 1", PassPercent: 50, MaxAttempts: 1, VisibleFrom: &from, VisibleUntil: &until,
		})
		require.NoError(t, err)
	}

	setWindow(time.Now().Add(24*time.Hour), time.Now().Add(48*time.Hour))
	_, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	assert.ErrorIs(t, err, quiz.ErrQuizClosed, "before window opens")

	setWindow(time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	_, err = f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	assert.ErrorIs(t, err, quiz.ErrQuizClosed, "after window closes")

	setWindow(time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	_, err = f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	assert.NoError(t, err)
}

func TestDraft_HiddenFromStudents(t *testing.T) {
	f := setup(t)
	dto, err := f.svc.CreateQuiz(f.course.Uuid, f.module.Uuid, "Draft quiz")
	require.NoError(t, err)

	_, err = f.svc.StartAttempt(f.course.Uuid, uuid.MustParse(dto.Uuid), f.student.Uuid)
	assert.ErrorIs(t, err, quiz.ErrQuizNotFound)

	student, err := f.svc.ListAssignments(f.course.Uuid, f.student.Uuid, false)
	require.NoError(t, err)
	assert.Empty(t, student)

	teacher, err := f.svc.ListAssignments(f.course.Uuid, f.teacher.Uuid, true)
	require.NoError(t, err)
	assert.Len(t, teacher, 1)
}

func TestPublish_NeedsQuestions(t *testing.T) {
	f := setup(t)
	dto, err := f.svc.CreateQuiz(f.course.Uuid, f.module.Uuid, "Empty")
	require.NoError(t, err)
	_, err = f.svc.Publish(f.course.Uuid, uuid.MustParse(dto.Uuid))
	assert.ErrorIs(t, err, quiz.ErrNoQuestions)
}

func TestEdit_ResetsAttempts(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	qu := uuid.MustParse(q.Uuid)

	_, err := f.attempt(t, q.Uuid, []int{0})
	require.NoError(t, err)

	_, err = f.svc.ReplaceQuestions(f.course.Uuid, qu, quiz.ReplaceQuestionsRequest{
		Questions: []quiz.QuestionInput{trueFalseQuestion, animalQuestion},
	})
	require.NoError(t, err)

	var attempts, answers int64
	f.db.Model(&quiz.Attempt{}).Count(&attempts)
	f.db.Model(&quiz.Answer{}).Count(&answers)
	assert.Zero(t, attempts)
	assert.Zero(t, answers)

	// student can take it again even though max attempts is 1
	_, err = f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	assert.NoError(t, err)
}

func TestQuestions_KeepTeacherOrder(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)
	assert.Equal(t, "Which animal can fly?", q.Questions[0].Prompt)
	assert.Equal(t, "Go is compiled", q.Questions[1].Prompt)

	reordered, err := f.svc.ReplaceQuestions(f.course.Uuid, uuid.MustParse(q.Uuid), quiz.ReplaceQuestionsRequest{
		Questions: []quiz.QuestionInput{trueFalseQuestion, animalQuestion},
	})
	require.NoError(t, err)
	assert.Equal(t, "Go is compiled", reordered.Questions[0].Prompt)
}

func TestQuestions_Validation(t *testing.T) {
	f := setup(t)
	dto, err := f.svc.CreateQuiz(f.course.Uuid, f.module.Uuid, "Quiz")
	require.NoError(t, err)
	qu := uuid.MustParse(dto.Uuid)

	bad := []quiz.QuestionInput{
		{Type: content.TypeMultipleChoice, Prompt: "p", Options: []string{"only one"}, Correct: []int{0}},
		{Type: content.TypeMultipleChoice, Prompt: "p", Options: []string{"a", "b"}, Correct: nil},
		{Type: content.TypeMultipleChoice, Prompt: "p", Options: []string{"a", "b"}, Correct: []int{2}},
		{Type: content.TypeMultipleChoice, Prompt: "p", Options: []string{"a", " "}, Correct: []int{0}},
		{Type: content.TypeTrueFalse, Prompt: "p", Correct: []int{0, 1}},
		{Type: content.TypeOrdering, Prompt: "p"},
	}
	for _, in := range bad {
		_, err := f.svc.ReplaceQuestions(f.course.Uuid, qu, quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{in}})
		assert.ErrorIs(t, err, quiz.ErrInvalidQuestion, "%+v", in)
	}
}

func TestStudentPayloads_NeverContainAnswerKey(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion, trueFalseQuestion)
	qu := uuid.MustParse(q.Uuid)

	attemptDto, err := f.svc.StartAttempt(f.course.Uuid, qu, f.student.Uuid)
	require.NoError(t, err)
	raw, err := json.Marshal(attemptDto)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "correct")
	assert.NotContains(t, string(raw), "answerKey")

	res, err := f.attempt(t, q.Uuid, []int{1, 2}, []int{0})
	require.NoError(t, err)
	raw, err = json.Marshal(res)
	require.NoError(t, err)
	assert.JSONEq(t, `{"percent":100,"passed":true,"attemptsUsed":1,"maxAttempts":1}`, string(raw))

	list, err := f.svc.ListAssignments(f.course.Uuid, f.student.Uuid, false)
	require.NoError(t, err)
	raw, err = json.Marshal(list)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "correct")
}

func TestResults_ListsStudentsWithBestScore(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)

	rows, err := f.svc.Results(f.course.Uuid, uuid.MustParse(q.Uuid))
	require.NoError(t, err)
	require.Len(t, rows, 1, "teacher is not listed")
	assert.Nil(t, rows[0].Percent)
	assert.Zero(t, rows[0].Attempts)

	_, err = f.attempt(t, q.Uuid, []int{0})
	require.NoError(t, err)

	rows, err = f.svc.Results(f.course.Uuid, uuid.MustParse(q.Uuid))
	require.NoError(t, err)
	require.NotNil(t, rows[0].Percent)
	assert.InDelta(t, 100.0, *rows[0].Percent, 0.001)
	assert.True(t, *rows[0].Passed)
	assert.Equal(t, 1, rows[0].Attempts)
}

func TestQuiz_NotFoundInOtherCourse(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	_, err := f.svc.GetEditQuiz(uuid.New(), uuid.MustParse(q.Uuid))
	assert.ErrorIs(t, err, quiz.ErrQuizNotFound)
}

func TestDeleteQuiz_RemovesEverything(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	_, err := f.attempt(t, q.Uuid, []int{0})
	require.NoError(t, err)

	require.NoError(t, f.svc.DeleteQuiz(f.course.Uuid, uuid.MustParse(q.Uuid)))

	_, err = f.svc.GetEditQuiz(f.course.Uuid, uuid.MustParse(q.Uuid))
	assert.ErrorIs(t, err, quiz.ErrQuizNotFound)
	var n int64
	f.db.Model(&quiz.Attempt{}).Count(&n)
	assert.Zero(t, n)
}

func TestListAssignments_IncludesHandinMaxPoints(t *testing.T) {
	f := setup(t)
	_ = f.publishedQuiz(t, trueFalseQuestion)

	require.NoError(t, f.db.Exec(`
		CREATE TABLE IF NOT EXISTS individual_submissions (id INTEGER PRIMARY KEY, content_id INTEGER, user_id INTEGER, deleted_at DATETIME);
		CREATE TABLE IF NOT EXISTS group_submissions (id INTEGER PRIMARY KEY, content_id INTEGER, deleted_at DATETIME);
		CREATE TABLE IF NOT EXISTS group_submission_members (id INTEGER PRIMARY KEY, group_submission_id INTEGER, user_id INTEGER);
	`).Error)

	handin := &content.Content{
		ModuleID: f.module.ID,
		Type:     content.TypeAssignment,
		Title:    "Hand-in Essay",
		Position: 2,
		Assignment: &content.Assignment{
			Type:      content.AssignmentHandIn,
			MaxPoints: 45,
		},
	}
	require.NoError(t, f.db.Create(handin).Error)

	// One active attempt and one soft-deleted attempt for the student.
	require.NoError(t, f.db.Exec("INSERT INTO individual_submissions (id, content_id, user_id, deleted_at) VALUES (1, ?, ?, NULL), (2, ?, ?, CURRENT_TIMESTAMP)", handin.ID, f.student.ID, handin.ID, f.student.ID).Error)

	items, err := f.svc.ListAssignments(f.course.Uuid, f.student.Uuid, false)
	require.NoError(t, err)
	require.Len(t, items, 2)

	byTitle := map[string]quiz.AssignmentDto{}
	for _, item := range items {
		byTitle[item.Title] = item
	}
	assert.Equal(t, 45.0, byTitle["Hand-in Essay"].MaxPoints)
	assert.Equal(t, 1, byTitle["Hand-in Essay"].AttemptsUsed)
	assert.Equal(t, 100.0, byTitle["Quiz 1"].MaxPoints)
}
