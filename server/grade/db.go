package grade

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrScoreOutOfRange    = errors.New("score exceeds assignment max points")
	ErrNotHandin          = errors.New("only hand-in assignments can be graded manually")
	ErrContentNotFound    = errors.New("assignment not found")
	ErrGraderNotFound     = errors.New("grader user not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrNotTeacher         = errors.New("user is not a teacher")
)

type userContentKey struct {
	userID    uint
	contentID uint
}

type submissionEntry struct {
	sub       submission.Submission
	grade     *Grade
	groupName string
}

type courseScoreLookup struct {
	quizBest         map[userContentKey]quiz.Attempt
	quizAttemptCount map[userContentKey]int
	handinSubs       map[userContentKey][]submissionEntry
	groupHandinSubs  map[userContentKey][]submissionEntry
}

func roundPercent2(pct float64) float64 {
	return math.Round(pct*100) / 100
}

func isCountedAssignment(cnt content.Content) bool {
	if cnt.Assignment == nil {
		return false
	}
	if cnt.Assignment.Type == content.AssignmentQuiz {
		return cnt.Assignment.Quiz != nil && cnt.Assignment.Quiz.Published
	}
	return true
}

func (s *GradeService) findUserByUUID(userUuid uuid.UUID) (*user.User, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *GradeService) loadGradesBySubmissionID(subIDs []uint, isGroup bool, preloadGrader bool) (map[uint]*Grade, error) {
	result := make(map[uint]*Grade, len(subIDs))
	if len(subIDs) == 0 {
		return result, nil
	}

	col := "individual_submission_id"
	if isGroup {
		col = "group_submission_id"
	}

	query := s.db.Where(col+" IN ?", subIDs)
	if preloadGrader {
		query = query.Preload("GradedBy")
	}

	var grades []Grade
	if err := query.Find(&grades).Error; err != nil {
		return nil, err
	}
	for i := range grades {
		var key uint
		if isGroup && grades[i].GroupSubmissionID != nil {
			key = *grades[i].GroupSubmissionID
		} else if !isGroup && grades[i].IndividualSubmissionID != nil {
			key = *grades[i].IndividualSubmissionID
		}
		if key != 0 {
			result[key] = &grades[i]
		}
	}
	return result, nil
}

func (s *GradeService) buildCourseScoreLookup(studentIDs []uint, contents []content.Content, preloadGrader bool) (*courseScoreLookup, error) {
	lookup := &courseScoreLookup{
		quizBest:         make(map[userContentKey]quiz.Attempt),
		quizAttemptCount: make(map[userContentKey]int),
		handinSubs:       make(map[userContentKey][]submissionEntry),
		groupHandinSubs:  make(map[userContentKey][]submissionEntry),
	}
	if len(studentIDs) == 0 || len(contents) == 0 {
		return lookup, nil
	}

	studentIDSet := make(map[uint]bool, len(studentIDs))
	for _, id := range studentIDs {
		studentIDSet[id] = true
	}

	var quizIDs []uint
	var handinIDs []uint
	var groupHandinIDs []uint
	for _, cnt := range contents {
		if !isCountedAssignment(cnt) {
			continue
		}
		if cnt.Assignment.Type == content.AssignmentQuiz {
			quizIDs = append(quizIDs, cnt.ID)
		} else if cnt.Assignment.AllowsGroupSubmission {
			groupHandinIDs = append(groupHandinIDs, cnt.ID)
		} else {
			handinIDs = append(handinIDs, cnt.ID)
		}
	}

	if len(quizIDs) > 0 {
		var attempts []quiz.Attempt
		if err := s.db.Where("user_id IN ? AND quiz_id IN ? AND submitted_at IS NOT NULL", studentIDs, quizIDs).
			Order("user_id ASC, quiz_id ASC, percent DESC, id DESC").
			Find(&attempts).Error; err != nil {
			return nil, err
		}
		for _, att := range attempts {
			key := userContentKey{userID: att.UserID, contentID: att.QuizID}
			lookup.quizAttemptCount[key]++
			if _, exists := lookup.quizBest[key]; !exists {
				lookup.quizBest[key] = att
			}
		}
	}

	if len(handinIDs) > 0 {
		var subs []submission.IndividualSubmission
		if err := s.db.
			Where("user_id IN ? AND content_id IN ?", studentIDs, handinIDs).
			Order("attempt DESC").
			Find(&subs).Error; err != nil {
			return nil, err
		}
		indIDs := make([]uint, 0, len(subs))
		for _, sub := range subs {
			indIDs = append(indIDs, sub.ID)
		}
		gradesBySubID, err := s.loadGradesBySubmissionID(indIDs, false, preloadGrader)
		if err != nil {
			return nil, err
		}
		for _, sub := range subs {
			if sub.UserID == nil {
				continue
			}
			key := userContentKey{userID: *sub.UserID, contentID: sub.ContentID}
			lookup.handinSubs[key] = append(lookup.handinSubs[key], submissionEntry{
				sub:   sub.Submission,
				grade: gradesBySubID[sub.ID],
			})
		}
	}

	if len(groupHandinIDs) > 0 {
		var grpSubs []submission.GroupSubmission
		if err := s.db.Preload("Members").Preload("Group").
			Where("content_id IN ?", groupHandinIDs).
			Order("attempt DESC").
			Find(&grpSubs).Error; err != nil {
			return nil, err
		}
		grpIDs := make([]uint, 0, len(grpSubs))
		for _, sub := range grpSubs {
			grpIDs = append(grpIDs, sub.ID)
		}
		gradesBySubID, err := s.loadGradesBySubmissionID(grpIDs, true, preloadGrader)
		if err != nil {
			return nil, err
		}
		for _, sub := range grpSubs {
			entry := submissionEntry{
				sub:   sub.Submission,
				grade: gradesBySubID[sub.ID],
			}
			if sub.Group != nil {
				entry.groupName = sub.Group.Name
			}
			seen := make(map[uint]bool, len(sub.Members)+1)
			for _, m := range sub.Members {
				if studentIDSet[m.ID] && !seen[m.ID] {
					seen[m.ID] = true
					key := userContentKey{userID: m.ID, contentID: sub.ContentID}
					lookup.groupHandinSubs[key] = append(lookup.groupHandinSubs[key], entry)
				}
			}
			if sub.UserID != nil && studentIDSet[*sub.UserID] && !seen[*sub.UserID] {
				key := userContentKey{userID: *sub.UserID, contentID: sub.ContentID}
				lookup.groupHandinSubs[key] = append(lookup.groupHandinSubs[key], entry)
			}
		}
	}

	return lookup, nil
}

// GradeLookup implements submission.IGradeLookup so submission can hydrate grades without importing grade.
type GradeLookup struct {
	db *gorm.DB
}

// NewGradeLookup constructs a submission.IGradeLookup backed by GORM.
func NewGradeLookup(db *gorm.DB) submission.IGradeLookup {
	return &GradeLookup{db: db}
}

// GetSubmissionGrades resolves grade summaries keyed by individual or group submission ID.
func (l *GradeLookup) GetSubmissionGrades(submissionIDs []uint, isGroup bool, visibleOnly bool) (map[uint]*submission.SubmissionGradeDto, error) {
	result := make(map[uint]*submission.SubmissionGradeDto, len(submissionIDs))
	if len(submissionIDs) == 0 {
		return result, nil
	}

	col := "individual_submission_id"
	if isGroup {
		col = "group_submission_id"
	}

	query := l.db.Where(col+" IN ?", submissionIDs)
	if visibleOnly {
		query = query.Where("is_visible_to_student = ?", true)
	}

	var grades []Grade
	if err := query.Find(&grades).Error; err != nil {
		return nil, err
	}

	for i := range grades {
		g := &grades[i]
		var key uint
		if isGroup && g.GroupSubmissionID != nil {
			key = *g.GroupSubmissionID
		} else if !isGroup && g.IndividualSubmissionID != nil {
			key = *g.IndividualSubmissionID
		}
		if key != 0 {
			result[key] = &submission.SubmissionGradeDto{
				Score:              g.Score,
				Feedback:           g.Feedback,
				IsVisibleToStudent: g.IsVisibleToStudent,
				GradedAt:           g.UpdatedAt,
				ReleasedAt:         g.ReleasedAt,
			}
		}
	}
	return result, nil
}

type IGradeService interface {
	UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto UpsertGradeDto) (*GradeDto, error)
	ReleaseAssignmentGrades(courseUuid, contentUuid uuid.UUID) (int, error)
	ListPending(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]PendingSubmissionDto, error)
	ListMyGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]StudentGradeDto, error)
	GetStudentFinalGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]CourseFinalGradeDto, error)
	GetMyCourseGradeOverview(userUuid, courseUuid uuid.UUID) (*CourseGradeOverviewDto, error)
	GetCourseRosterFinalGrades(courseUuid uuid.UUID) ([]StudentFinalGradeRowDto, error)
	ExportCourseCSV(courseUuid uuid.UUID, w io.Writer) error
}

type GradeService struct {
	db            *gorm.DB
	notifications notification.INotifier
	logger        *zap.SugaredLogger
}

func NewGradeService(db *gorm.DB, notifications notification.INotifier, logger *zap.SugaredLogger) IGradeService {
	return &GradeService{
		db:            db,
		notifications: notifications,
		logger:        logger,
	}
}

// UpsertGrade creates or updates a manual grade for an individual or group submission.
func (s *GradeService) UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto UpsertGradeDto) (*GradeDto, error) {
	var grader user.User
	if err := s.db.Where("uuid = ?", graderUuid).First(&grader).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGraderNotFound
		}
		return nil, err
	}

	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Joins("JOIN courses ON courses.id = modules.course_id AND courses.deleted_at IS NULL").
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	if cnt.Assignment == nil || cnt.Assignment.Type != content.AssignmentHandIn {
		return nil, ErrNotHandin
	}

	isGroup := cnt.Assignment.AllowsGroupSubmission
	var targetSubID uint
	var filterCol string
	var recipientUuids []uuid.UUID

	if isGroup {
		var grpSub submission.GroupSubmission
		err = s.db.
			Preload("Members").
			Where("uuid = ? AND content_id = ?", submissionUuid, cnt.ID).
			First(&grpSub).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrSubmissionNotFound
			}
			return nil, err
		}
		targetSubID = grpSub.ID
		filterCol = "group_submission_id"

		seen := make(map[uuid.UUID]bool, len(grpSub.Members)+1)
		for _, m := range grpSub.Members {
			if m.Uuid != uuid.Nil && !seen[m.Uuid] {
				seen[m.Uuid] = true
				recipientUuids = append(recipientUuids, m.Uuid)
			}
		}
		if grpSub.UserID != nil {
			var submitter user.User
			if err := s.db.Select("uuid").Where("id = ?", *grpSub.UserID).First(&submitter).Error; err == nil {
				if submitter.Uuid != uuid.Nil && !seen[submitter.Uuid] {
					recipientUuids = append(recipientUuids, submitter.Uuid)
				}
			}
		}
	} else {
		var indSub submission.IndividualSubmission
		err = s.db.
			Where("uuid = ? AND content_id = ?", submissionUuid, cnt.ID).
			First(&indSub).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrSubmissionNotFound
			}
			return nil, err
		}
		targetSubID = indSub.ID
		filterCol = "individual_submission_id"

		if indSub.UserID != nil {
			var student user.User
			if err := s.db.Select("uuid").Where("id = ?", *indSub.UserID).First(&student).Error; err == nil {
				if student.Uuid != uuid.Nil {
					recipientUuids = append(recipientUuids, student.Uuid)
				}
			}
		}
	}

	now := time.Now()
	var shouldNotify bool
	var savedGrade Grade
	var maxPoints float64

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock assignment row to prevent concurrent maxPoints reduction below this grade.
		var lockedAsgn content.Assignment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("content_id = ?", cnt.ID).First(&lockedAsgn).Error; err != nil {
			return err
		}
		maxPoints = lockedAsgn.MaxPoints
		if maxPoints <= 0 {
			maxPoints = content.DefaultHandinMaxPoints
		}
		if math.IsNaN(*dto.Score) || *dto.Score < 0 || *dto.Score > maxPoints {
			return ErrScoreOutOfRange
		}

		var existing Grade
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(filterCol+" = ?", targetSubID).
			First(&existing).Error

		if findErr == nil {
			wasVisible := existing.IsVisibleToStudent
			existing.Score = *dto.Score
			existing.Feedback = dto.Feedback
			existing.IsVisibleToStudent = dto.IsVisibleToStudent
			existing.GradedByID = &grader.ID

			if !wasVisible && dto.IsVisibleToStudent {
				existing.ReleasedAt = &now
				shouldNotify = true
			} else if !dto.IsVisibleToStudent {
				existing.ReleasedAt = nil
			}

			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			savedGrade = existing
			return nil
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			var relAt *time.Time
			if dto.IsVisibleToStudent {
				relAt = &now
				shouldNotify = true
			}
			newGrade := Grade{
				Score:              *dto.Score,
				Feedback:           dto.Feedback,
				IsVisibleToStudent: dto.IsVisibleToStudent,
				ReleasedAt:         relAt,
				GradedByID:         &grader.ID,
			}
			if isGroup {
				newGrade.GroupSubmissionID = &targetSubID
			} else {
				newGrade.IndividualSubmissionID = &targetSubID
			}
			if err := tx.Create(&newGrade).Error; err != nil {
				return err
			}
			savedGrade = newGrade
			return nil
		}
		return findErr
	})
	if err != nil {
		s.logger.Errorf("Failed to save grade for submission %s: %v", submissionUuid, err)
		return nil, err
	}

	if shouldNotify && len(recipientUuids) > 0 {
		msg := notification.NotificationContent{
			Title: fmt.Sprintf("Grade Published: %s", cnt.Title),
			Body:  fmt.Sprintf("Your submission for %q has been graded (%g/%g).", cnt.Title, savedGrade.Score, maxPoints),
			URL:   "/grades",
		}
		if isGroup {
			msg.Body = fmt.Sprintf("Your group submission for %q has been graded (%g/%g).", cnt.Title, savedGrade.Score, maxPoints)
			if _, nErr := s.notifications.NotifySystemMany(recipientUuids, msg); nErr != nil {
				s.logger.Warnf("Failed to notify group members of grade release: %v", nErr)
			}
		} else {
			if _, nErr := s.notifications.NotifySystem(recipientUuids[0], msg); nErr != nil {
				s.logger.Warnf("Failed to notify student %s of grade release: %v", recipientUuids[0], nErr)
			}
		}
	}

	return &GradeDto{
		Score:              savedGrade.Score,
		MaxPoints:          maxPoints,
		Feedback:           savedGrade.Feedback,
		IsVisibleToStudent: savedGrade.IsVisibleToStudent,
		GradedByEmail:      grader.Email,
		GradedAt:           savedGrade.UpdatedAt,
		ReleasedAt:         savedGrade.ReleasedAt,
	}, nil
}

// ReleaseAssignmentGrades releases all unreleased grades for an individual or group assignment.
func (s *GradeService) ReleaseAssignmentGrades(courseUuid, contentUuid uuid.UUID) (int, error) {
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Joins("JOIN courses ON courses.id = modules.course_id AND courses.deleted_at IS NULL").
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrContentNotFound
		}
		return 0, err
	}

	if cnt.Assignment == nil || cnt.Assignment.Type != content.AssignmentHandIn {
		return 0, ErrNotHandin
	}

	var gradeIDs []uint
	userUuidMap := make(map[uuid.UUID]bool)

	if cnt.Assignment.AllowsGroupSubmission {
		type groupGradeRow struct {
			GradeID           uint `gorm:"column:grade_id"`
			GroupSubmissionID uint `gorm:"column:group_submission_id"`
		}
		var rows []groupGradeRow
		err = s.db.Table("grades").
			Select("grades.id as grade_id, group_submissions.id as group_submission_id").
			Joins("JOIN group_submissions ON group_submissions.id = grades.group_submission_id AND group_submissions.deleted_at IS NULL").
			Where("group_submissions.content_id = ? AND grades.is_visible_to_student = ? AND grades.deleted_at IS NULL", cnt.ID, false).
			Scan(&rows).Error
		if err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			return 0, nil
		}

		subIDs := make([]uint, 0, len(rows))
		gradeIDs = make([]uint, 0, len(rows))
		for _, r := range rows {
			gradeIDs = append(gradeIDs, r.GradeID)
			subIDs = append(subIDs, r.GroupSubmissionID)
		}

		var memberUuids []uuid.UUID
		if err := s.db.Table("group_submission_members gsm").
			Joins("JOIN users ON users.id = gsm.user_id AND users.deleted_at IS NULL").
			Where("gsm.group_submission_id IN ?", subIDs).
			Pluck("users.uuid", &memberUuids).Error; err != nil {
			return 0, err
		}
		for _, u := range memberUuids {
			if u != uuid.Nil {
				userUuidMap[u] = true
			}
		}

		var submitterUuids []uuid.UUID
		if err := s.db.Table("group_submissions gs").
			Joins("JOIN users ON users.id = gs.user_id AND users.deleted_at IS NULL").
			Where("gs.id IN ?", subIDs).
			Pluck("users.uuid", &submitterUuids).Error; err != nil {
			return 0, err
		}
		for _, u := range submitterUuids {
			if u != uuid.Nil {
				userUuidMap[u] = true
			}
		}
	} else {
		type unreleasedRow struct {
			GradeID uint      `gorm:"column:grade_id"`
			UserID  uint      `gorm:"column:user_id"`
			UserUid uuid.UUID `gorm:"column:user_uuid"`
		}

		var rows []unreleasedRow
		err = s.db.Table("grades").
			Select("grades.id as grade_id, users.id as user_id, users.uuid as user_uuid").
			Joins("JOIN individual_submissions ON individual_submissions.id = grades.individual_submission_id AND individual_submissions.deleted_at IS NULL").
			Joins("JOIN users ON users.id = individual_submissions.user_id AND users.deleted_at IS NULL").
			Where("individual_submissions.content_id = ? AND grades.is_visible_to_student = ? AND grades.deleted_at IS NULL", cnt.ID, false).
			Scan(&rows).Error
		if err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			return 0, nil
		}

		gradeIDs = make([]uint, 0, len(rows))
		for _, r := range rows {
			gradeIDs = append(gradeIDs, r.GradeID)
			if r.UserUid != uuid.Nil {
				userUuidMap[r.UserUid] = true
			}
		}
	}

	now := time.Now()
	err = s.db.Model(&Grade{}).
		Where("id IN ?", gradeIDs).
		Updates(map[string]any{
			"is_visible_to_student": true,
			"released_at":           now,
		}).Error
	if err != nil {
		s.logger.Errorf("Failed to release grades for assignment %s: %v", contentUuid, err)
		return 0, err
	}

	if len(userUuidMap) > 0 {
		userUuids := make([]uuid.UUID, 0, len(userUuidMap))
		for u := range userUuidMap {
			userUuids = append(userUuids, u)
		}
		_, nErr := s.notifications.NotifySystemMany(userUuids, notification.NotificationContent{
			Title: fmt.Sprintf("Grades Published: %s", cnt.Title),
			Body:  fmt.Sprintf("Grades for %q have been published.", cnt.Title),
			URL:   "/grades",
		})
		if nErr != nil {
			s.logger.Warnf("Failed to send bulk grade release notifications: %v", nErr)
		}
	}

	return len(gradeIDs), nil
}

// ListPending returns latest ungraded individual and group submissions across courses taught by the user.
func (s *GradeService) ListPending(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]PendingSubmissionDto, error) {
	u, err := s.findUserByUUID(userUuid)
	if err != nil {
		return nil, err
	}

	var courseIDs []uint
	courseIDSet := make(map[uint]bool)
	if !isAdmin {
		err := s.db.Model(&course.Enrollment{}).
			Where("user_id = ? AND role = ? AND deleted_at IS NULL", u.ID, course.RoleTeacher).
			Pluck("course_id", &courseIDs).Error
		if err != nil {
			return nil, err
		}
		if len(courseIDs) == 0 {
			return nil, ErrNotTeacher
		}
		for _, id := range courseIDs {
			courseIDSet[id] = true
		}
	}

	if courseUuid != nil {
		var targetCourse course.Course
		if err := s.db.Select("id").Where("uuid = ?", *courseUuid).First(&targetCourse).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, course.ErrCourseNotFound
			}
			return nil, err
		}
		if !isAdmin && !courseIDSet[targetCourse.ID] {
			return nil, ErrNotTeacher
		}
	}

	type queryResult struct {
		SubmissionID    uint       `gorm:"column:submission_id"`
		SubmissionUuid  uuid.UUID  `gorm:"column:submission_uuid"`
		CourseUuid      uuid.UUID  `gorm:"column:course_uuid"`
		ContentUuid     uuid.UUID  `gorm:"column:content_uuid"`
		CourseTitle     string     `gorm:"column:course_title"`
		CourseTerm      string     `gorm:"column:course_term"`
		CourseYear      int        `gorm:"column:course_year"`
		AssignmentTitle string     `gorm:"column:assignment_title"`
		UserEmail       string     `gorm:"column:user_email"`
		UserName        string     `gorm:"column:user_name"`
		GroupName       string     `gorm:"column:group_name"`
		Attempt         int        `gorm:"column:attempt"`
		IsLate          bool       `gorm:"column:is_late"`
		SubmittedAt     time.Time  `gorm:"column:submitted_at"`
		Deadline        *time.Time `gorm:"column:deadline"`
		MaxPoints       float64    `gorm:"column:max_points"`
	}

	indQuery := s.db.Table("individual_submissions s").
		Select(`s.id as submission_id,
			s.uuid as submission_uuid,
			crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			cnt.title as assignment_title,
			s.user_email as user_email,
			COALESCE(u.username, '') as user_name,
			'' as group_name,
			s.attempt as attempt,
			s.is_late as is_late,
			s.submitted_at as submitted_at,
			asgn.deadline as deadline,
			COALESCE(asgn.max_points, 5) as max_points`).
		Joins("JOIN contents cnt ON cnt.id = s.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN grades g ON g.individual_submission_id = s.id AND g.deleted_at IS NULL").
		Where("asgn.type = ? AND asgn.allows_group_submission = ?", content.AssignmentHandIn, false).
		Where("g.id IS NULL").
		Where("s.deleted_at IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM individual_submissions s2 WHERE s2.user_id = s.user_id AND s2.content_id = s.content_id AND s2.attempt > s.attempt AND s2.deleted_at IS NULL)")

	grpQuery := s.db.Table("group_submissions s").
		Select(`s.id as submission_id,
			s.uuid as submission_uuid,
			crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			cnt.title as assignment_title,
			s.user_email as user_email,
			COALESCE(u.username, '') as user_name,
			COALESCE(grp.name, '') as group_name,
			s.attempt as attempt,
			s.is_late as is_late,
			s.submitted_at as submitted_at,
			asgn.deadline as deadline,
			COALESCE(asgn.max_points, 5) as max_points`).
		Joins("JOIN contents cnt ON cnt.id = s.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Joins("LEFT JOIN groups grp ON grp.id = s.group_id").
		Joins("LEFT JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN grades g ON g.group_submission_id = s.id AND g.deleted_at IS NULL").
		Where("asgn.type = ? AND asgn.allows_group_submission = ?", content.AssignmentHandIn, true).
		Where("g.id IS NULL").
		Where("s.deleted_at IS NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM group_submissions s2
			WHERE s2.content_id = s.content_id
			  AND ((s.group_id IS NOT NULL AND s2.group_id = s.group_id) OR (s.group_id IS NULL AND s2.group_id IS NULL AND s2.user_id = s.user_id))
			  AND s2.attempt > s.attempt AND s2.deleted_at IS NULL
		)`)

	if !isAdmin {
		indQuery = indQuery.Where("crs.id IN ?", courseIDs)
		grpQuery = grpQuery.Where("crs.id IN ?", courseIDs)
	}
	if courseUuid != nil {
		indQuery = indQuery.Where("crs.uuid = ?", *courseUuid)
		grpQuery = grpQuery.Where("crs.uuid = ?", *courseUuid)
	}

	var indResults []queryResult
	if err := indQuery.Order("s.submitted_at ASC").Limit(500).Scan(&indResults).Error; err != nil {
		return nil, err
	}

	var grpResults []queryResult
	if err := grpQuery.Order("s.submitted_at ASC").Limit(500).Scan(&grpResults).Error; err != nil {
		return nil, err
	}

	memberEmailsBySubID := make(map[uint][]string, len(grpResults))
	if len(grpResults) > 0 {
		grpSubIDs := make([]uint, 0, len(grpResults))
		for _, r := range grpResults {
			grpSubIDs = append(grpSubIDs, r.SubmissionID)
		}
		type memberEmailRow struct {
			GroupSubmissionID uint   `gorm:"column:group_submission_id"`
			Email             string `gorm:"column:email"`
		}
		var memberRows []memberEmailRow
		if err := s.db.Table("group_submission_members gsm").
			Select("gsm.group_submission_id, u.email").
			Joins("JOIN users u ON u.id = gsm.user_id AND u.deleted_at IS NULL").
			Where("gsm.group_submission_id IN ?", grpSubIDs).
			Order("u.email ASC").
			Scan(&memberRows).Error; err != nil {
			return nil, err
		}
		for _, mr := range memberRows {
			memberEmailsBySubID[mr.GroupSubmissionID] = append(memberEmailsBySubID[mr.GroupSubmissionID], mr.Email)
		}
	}

	toPendingDto := func(r queryResult, isGroup bool) PendingSubmissionDto {
		maxPts := r.MaxPoints
		if maxPts <= 0 {
			maxPts = content.DefaultHandinMaxPoints
		}
		var memberEmails []string
		if isGroup {
			memberEmails = memberEmailsBySubID[r.SubmissionID]
		}
		return PendingSubmissionDto{
			SubmissionUuid:  r.SubmissionUuid,
			CourseUuid:      r.CourseUuid,
			ContentUuid:     r.ContentUuid,
			CourseCode:      fmt.Sprintf("%s %d", r.CourseTerm, r.CourseYear),
			CourseName:      r.CourseTitle,
			AssignmentTitle: r.AssignmentTitle,
			UserEmail:       r.UserEmail,
			UserName:        r.UserName,
			IsGroup:         isGroup,
			GroupName:       r.GroupName,
			MemberEmails:    memberEmails,
			Attempt:         r.Attempt,
			IsLate:          r.IsLate,
			SubmittedAt:     r.SubmittedAt,
			Deadline:        r.Deadline,
			MaxPoints:       maxPts,
		}
	}

	dtos := make([]PendingSubmissionDto, 0, len(indResults)+len(grpResults))
	for _, r := range indResults {
		dtos = append(dtos, toPendingDto(r, false))
	}
	for _, r := range grpResults {
		dtos = append(dtos, toPendingDto(r, true))
	}

	sort.Slice(dtos, func(i, j int) bool {
		if !dtos[i].SubmittedAt.Equal(dtos[j].SubmittedAt) {
			return dtos[i].SubmittedAt.Before(dtos[j].SubmittedAt)
		}
		return dtos[i].SubmissionUuid.String() < dtos[j].SubmissionUuid.String()
	})
	if len(dtos) > 500 {
		dtos = dtos[:500]
	}

	return dtos, nil
}

// ListMyGrades returns the latest released hand-in grade per assignment and quiz best-attempt scores for the student.
func (s *GradeService) ListMyGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]StudentGradeDto, error) {
	u, err := s.findUserByUUID(userUuid)
	if err != nil {
		return nil, err
	}

	// 1. Fetch latest released hand-in grade per assignment (individual + group)
	type handinGradeRow struct {
		CourseUuid      uuid.UUID `gorm:"column:course_uuid"`
		ContentUuid     uuid.UUID `gorm:"column:content_uuid"`
		CourseTitle     string    `gorm:"column:course_title"`
		CourseTerm      string    `gorm:"column:course_term"`
		CourseYear      int       `gorm:"column:course_year"`
		ModuleTitle     string    `gorm:"column:module_title"`
		AssignmentTitle string    `gorm:"column:assignment_title"`
		GroupName       string    `gorm:"column:group_name"`
		ModPosition     int       `gorm:"column:mod_position"`
		CntPosition     int       `gorm:"column:cnt_position"`
		Attempt         int       `gorm:"column:attempt"`
		Score           float64   `gorm:"column:score"`
		MaxPoints       float64   `gorm:"column:max_points"`
		Feedback        string    `gorm:"column:feedback"`
		GradedAt        time.Time `gorm:"column:graded_at"`
		IsGroup         bool      `gorm:"-"`
	}

	indQuery := s.db.Table("grades g").
		Select(`crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			mod.title as module_title,
			cnt.title as assignment_title,
			'' as group_name,
			mod.position as mod_position,
			cnt.position as cnt_position,
			ind.attempt as attempt,
			g.score as score,
			COALESCE(asgn.max_points, 5) as max_points,
			g.feedback as feedback,
			g.updated_at as graded_at`).
		Joins("JOIN individual_submissions ind ON ind.id = g.individual_submission_id AND ind.deleted_at IS NULL").
		Joins("JOIN contents cnt ON cnt.id = ind.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Where("ind.user_id = ? AND g.is_visible_to_student = ? AND g.deleted_at IS NULL", u.ID, true).
		Where(`NOT EXISTS (
			SELECT 1 FROM individual_submissions ind2
			JOIN grades g2 ON g2.individual_submission_id = ind2.id AND g2.deleted_at IS NULL
			WHERE ind2.user_id = ind.user_id AND ind2.content_id = ind.content_id
			  AND ind2.attempt > ind.attempt AND ind2.deleted_at IS NULL
			  AND g2.is_visible_to_student = ?
		)`, true)

	grpQuery := s.db.Table("grades g").
		Select(`crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			mod.title as module_title,
			cnt.title as assignment_title,
			COALESCE(grp.name, '') as group_name,
			mod.position as mod_position,
			cnt.position as cnt_position,
			gs.attempt as attempt,
			g.score as score,
			COALESCE(asgn.max_points, 5) as max_points,
			g.feedback as feedback,
			g.updated_at as graded_at`).
		Joins("JOIN group_submissions gs ON gs.id = g.group_submission_id AND gs.deleted_at IS NULL").
		Joins("LEFT JOIN groups grp ON grp.id = gs.group_id").
		Joins("JOIN contents cnt ON cnt.id = gs.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Where("(gs.user_id = ? OR gs.id IN (SELECT group_submission_id FROM group_submission_members WHERE user_id = ?)) AND g.is_visible_to_student = ? AND g.deleted_at IS NULL", u.ID, u.ID, true).
		Where(`NOT EXISTS (
			SELECT 1 FROM group_submissions gs2
			JOIN grades g2 ON g2.group_submission_id = gs2.id AND g2.deleted_at IS NULL
			WHERE gs2.content_id = gs.content_id
			  AND (gs2.user_id = ? OR gs2.id IN (SELECT group_submission_id FROM group_submission_members WHERE user_id = ?))
			  AND gs2.attempt > gs.attempt AND gs2.deleted_at IS NULL
			  AND g2.is_visible_to_student = ?
		)`, u.ID, u.ID, true)

	if courseUuid != nil {
		indQuery = indQuery.Where("crs.uuid = ?", *courseUuid)
		grpQuery = grpQuery.Where("crs.uuid = ?", *courseUuid)
	}

	var indRows []handinGradeRow
	if err := indQuery.Order("crs.title ASC, mod.position ASC, cnt.position ASC").Scan(&indRows).Error; err != nil {
		return nil, err
	}

	var grpRows []handinGradeRow
	if err := grpQuery.Order("crs.title ASC, mod.position ASC, cnt.position ASC").Scan(&grpRows).Error; err != nil {
		return nil, err
	}
	for i := range grpRows {
		grpRows[i].IsGroup = true
	}

	allHandins := append(indRows, grpRows...)
	sort.SliceStable(allHandins, func(i, j int) bool {
		if allHandins[i].CourseTitle != allHandins[j].CourseTitle {
			return allHandins[i].CourseTitle < allHandins[j].CourseTitle
		}
		if allHandins[i].ModPosition != allHandins[j].ModPosition {
			return allHandins[i].ModPosition < allHandins[j].ModPosition
		}
		return allHandins[i].CntPosition < allHandins[j].CntPosition
	})

	dtos := make([]StudentGradeDto, 0, len(allHandins))
	for _, r := range allHandins {
		maxPts := r.MaxPoints
		if maxPts <= 0 {
			maxPts = content.DefaultHandinMaxPoints
		}
		pct := 0.0
		if maxPts > 0 {
			pct = (r.Score / maxPts) * 100.0
		}
		dtos = append(dtos, StudentGradeDto{
			Type:            GradeTypeHandin,
			CourseUuid:      r.CourseUuid,
			ContentUuid:     r.ContentUuid,
			CourseName:      r.CourseTitle,
			CourseCode:      fmt.Sprintf("%s %d", r.CourseTerm, r.CourseYear),
			ModuleTitle:     r.ModuleTitle,
			AssignmentTitle: r.AssignmentTitle,
			IsGroup:         r.IsGroup,
			GroupName:       r.GroupName,
			Attempt:         r.Attempt,
			Score:           r.Score,
			MaxPoints:       maxPts,
			Percent:         pct,
			Feedback:        r.Feedback,
			GradedAt:        r.GradedAt,
		})
	}

	// 2. Fetch published quiz best attempts
	type quizRow struct {
		ContentID       uint      `gorm:"column:content_id"`
		ContentUuid     uuid.UUID `gorm:"column:content_uuid"`
		CourseUuid      uuid.UUID `gorm:"column:course_uuid"`
		CourseTitle     string    `gorm:"column:course_title"`
		CourseTerm      string    `gorm:"column:course_term"`
		CourseYear      int       `gorm:"column:course_year"`
		ModuleTitle     string    `gorm:"column:module_title"`
		AssignmentTitle string    `gorm:"column:assignment_title"`
	}

	qQuery := s.db.Table("contents cnt").
		Select(`cnt.id as content_id,
			cnt.uuid as content_uuid,
			crs.uuid as course_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			mod.title as module_title,
			cnt.title as assignment_title`).
		Joins("JOIN quizzes q ON q.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Joins("JOIN enrollments enr ON enr.course_id = crs.id AND enr.user_id = ? AND enr.deleted_at IS NULL", u.ID).
		Where("cnt.deleted_at IS NULL AND q.published = ?", true)

	if courseUuid != nil {
		qQuery = qQuery.Where("crs.uuid = ?", *courseUuid)
	}

	var qRows []quizRow
	if err := qQuery.Order("crs.title ASC, mod.position ASC, cnt.position ASC").Scan(&qRows).Error; err != nil {
		return nil, err
	}

	if len(qRows) == 0 {
		return dtos, nil
	}

	quizIDs := make([]uint, 0, len(qRows))
	for _, qr := range qRows {
		quizIDs = append(quizIDs, qr.ContentID)
	}

	var attempts []quiz.Attempt
	if err := s.db.Where("user_id = ? AND quiz_id IN ? AND submitted_at IS NOT NULL", u.ID, quizIDs).
		Order("quiz_id ASC, percent DESC, id DESC").
		Find(&attempts).Error; err != nil {
		return nil, err
	}

	bestByQuiz := make(map[uint]quiz.Attempt, len(quizIDs))
	countByQuiz := make(map[uint]int, len(quizIDs))
	for _, att := range attempts {
		countByQuiz[att.QuizID]++
		if _, exists := bestByQuiz[att.QuizID]; !exists {
			bestByQuiz[att.QuizID] = att
		}
	}

	for _, qr := range qRows {
		best, ok := bestByQuiz[qr.ContentID]
		if !ok {
			continue
		}

		submittedTime := time.Now()
		if best.SubmittedAt != nil {
			submittedTime = *best.SubmittedAt
		}

		dtos = append(dtos, StudentGradeDto{
			Type:            GradeTypeQuiz,
			CourseUuid:      qr.CourseUuid,
			ContentUuid:     qr.ContentUuid,
			CourseName:      qr.CourseTitle,
			CourseCode:      fmt.Sprintf("%s %d", qr.CourseTerm, qr.CourseYear),
			ModuleTitle:     qr.ModuleTitle,
			AssignmentTitle: qr.AssignmentTitle,
			Attempt:         countByQuiz[qr.ContentID],
			Score:           best.Percent,
			MaxPoints:       100,
			Percent:         best.Percent,
			Feedback:        "",
			GradedAt:        submittedTime,
		})
	}

	return dtos, nil
}

// ExportCourseCSV exports the complete gradebook matrix for all students in a course.
func (s *GradeService) ExportCourseCSV(courseUuid uuid.UUID, w io.Writer) error {
	var crs course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&crs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return course.ErrCourseNotFound
		}
		return err
	}

	var enrollments []course.Enrollment
	err := s.db.Preload("User").
		Where("course_id = ? AND role = ? AND deleted_at IS NULL", crs.ID, course.RoleStudent).
		Order("user_email ASC").
		Find(&enrollments).Error
	if err != nil {
		return err
	}

	var contents []content.Content
	err = s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Preload("Module").
		Preload("Assignment").
		Preload("Assignment.Quiz").
		Where("modules.course_id = ? AND contents.type = ? AND contents.deleted_at IS NULL", crs.ID, content.TypeAssignment).
		Order("modules.position ASC, contents.position ASC").
		Find(&contents).Error
	if err != nil {
		return err
	}

	studentIDs := make([]uint, 0, len(enrollments))
	for _, enr := range enrollments {
		if enr.User != nil {
			studentIDs = append(studentIDs, enr.User.ID)
		}
	}

	lookup, err := s.buildCourseScoreLookup(studentIDs, contents, true)
	if err != nil {
		return err
	}

	records := make([]GradebookRecord, 0, len(enrollments)*len(contents))

	for _, enr := range enrollments {
		stUser := enr.User
		if stUser == nil {
			continue
		}

		finalScore, finalMax := computeStudentCourseTotal(stUser.ID, contents, lookup, false)
		finalPct := 0.0
		if finalMax > 0 {
			finalPct = roundPercent2((finalScore / finalMax) * 100.0)
		}
		finalGrade := crs.CalculateGrade(finalPct)

		for _, cnt := range contents {
			if !isCountedAssignment(cnt) {
				continue
			}

			modTitle := ""
			if cnt.Module != nil {
				modTitle = cnt.Module.Title
			}

			asgn := cnt.Assignment
			rec := GradebookRecord{
				StudentEmail:       enr.UserEmail,
				StudentName:        stUser.Username,
				ModuleTitle:        modTitle,
				AssignmentTitle:    cnt.Title,
				FinalCourseScore:   fmt.Sprintf("%.2f", finalScore),
				FinalCourseMax:     fmt.Sprintf("%.2f", finalMax),
				FinalCoursePercent: fmt.Sprintf("%.2f%%", finalPct),
				FinalCourseGrade:   finalGrade,
			}

			key := userContentKey{userID: stUser.ID, contentID: cnt.ID}

			if asgn.Type == content.AssignmentQuiz {
				rec.AssignmentType = "QUIZ"
				if best, ok := lookup.quizBest[key]; ok {
					rec.Attempt = fmt.Sprintf("%d", lookup.quizAttemptCount[key])
					if best.SubmittedAt != nil {
						rec.SubmittedAt = best.SubmittedAt.Format(time.RFC3339)
					}
					rec.Score = fmt.Sprintf("%.2f", best.Percent)
					rec.MaxPoints = "100.00"
					rec.Percent = fmt.Sprintf("%.2f%%", best.Percent)
					rec.IsVisibleToStudent = "true"
					rec.GradedAt = rec.SubmittedAt
				}
				records = append(records, rec)
				continue
			}

			maxPts := asgn.MaxPoints
			if maxPts <= 0 {
				maxPts = content.DefaultHandinMaxPoints
			}
			rec.MaxPoints = fmt.Sprintf("%.2f", maxPts)

			if asgn.AllowsGroupSubmission {
				rec.AssignmentType = "HANDIN (GROUP)"
			} else {
				rec.AssignmentType = "HANDIN"
			}

			entries := lookup.handinSubs[key]
			if asgn.AllowsGroupSubmission {
				entries = lookup.groupHandinSubs[key]
			}
			if len(entries) > 0 {
				var targetEntry *submissionEntry
				for i := range entries {
					if entries[i].grade != nil {
						targetEntry = &entries[i]
						break
					}
				}
				if targetEntry == nil {
					targetEntry = &entries[0]
				}
				populateHandinCSVRecord(&rec, targetEntry.sub, targetEntry.grade, maxPts)
			}

			records = append(records, rec)
		}
	}

	return WriteGradebookCSV(w, records)
}

func populateHandinCSVRecord(rec *GradebookRecord, sub submission.Submission, gr *Grade, maxPts float64) {
	rec.Attempt = fmt.Sprintf("%d", sub.Attempt)
	rec.SubmittedAt = sub.SubmittedAt.Format(time.RFC3339)
	if sub.IsLate {
		rec.IsLate = "true"
	} else {
		rec.IsLate = "false"
	}

	if gr != nil {
		rec.Score = fmt.Sprintf("%.2f", gr.Score)
		pct := 0.0
		if maxPts > 0 {
			pct = (gr.Score / maxPts) * 100.0
		}
		rec.Percent = fmt.Sprintf("%.2f%%", pct)
		if gr.IsVisibleToStudent {
			rec.IsVisibleToStudent = "true"
		} else {
			rec.IsVisibleToStudent = "false"
		}
		if gr.GradedBy != nil {
			rec.GradedByEmail = gr.GradedBy.Email
		}
		rec.GradedAt = gr.UpdatedAt.Format(time.RFC3339)
		rec.Feedback = gr.Feedback
	}
}

// computeStudentCourseTotal calculates the earned score and total possible score for a student across all published assignments in a course.
func computeStudentCourseTotal(studentID uint, contents []content.Content, lookup *courseScoreLookup, studentViewOnly bool) (earned float64, totalMax float64) {
	for _, cnt := range contents {
		if !isCountedAssignment(cnt) {
			continue
		}
		asgn := cnt.Assignment

		maxPts := asgn.MaxPoints
		if maxPts <= 0 {
			maxPts = content.DefaultHandinMaxPoints
		}

		key := userContentKey{userID: studentID, contentID: cnt.ID}

		if asgn.Type == content.AssignmentQuiz {
			totalMax += 100.0
			if best, ok := lookup.quizBest[key]; ok {
				earned += best.Percent
			}
			continue
		}

		totalMax += maxPts

		entries := lookup.handinSubs[key]
		if asgn.AllowsGroupSubmission {
			entries = lookup.groupHandinSubs[key]
		}
		for _, entry := range entries {
			if entry.grade != nil && (!studentViewOnly || entry.grade.IsVisibleToStudent) {
				earned += entry.grade.Score
				break
			}
		}
	}
	return earned, totalMax
}

// GetStudentFinalGrades calculates the student's cumulative subject evaluation and final letter grade across enrolled courses.
func (s *GradeService) GetStudentFinalGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]CourseFinalGradeDto, error) {
	u, err := s.findUserByUUID(userUuid)
	if err != nil {
		return nil, err
	}

	enrQuery := s.db.Preload("Course").
		Where("enrollments.user_id = ? AND enrollments.role = ? AND enrollments.deleted_at IS NULL", u.ID, course.RoleStudent)

	if courseUuid != nil {
		enrQuery = enrQuery.Joins("JOIN courses ON courses.id = enrollments.course_id AND courses.uuid = ? AND courses.deleted_at IS NULL", *courseUuid)
	}

	var enrollments []course.Enrollment
	if err := enrQuery.Find(&enrollments).Error; err != nil {
		return nil, err
	}

	courseIDs := make([]uint, 0, len(enrollments))
	for _, enr := range enrollments {
		if enr.Course != nil {
			courseIDs = append(courseIDs, enr.Course.ID)
		}
	}

	contentsByCourse := make(map[uint][]content.Content, len(courseIDs))
	var allContents []content.Content
	if len(courseIDs) > 0 {
		err := s.db.
			Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
			Preload("Module").
			Preload("Assignment").
			Preload("Assignment.Quiz").
			Where("modules.course_id IN ? AND contents.type = ? AND contents.deleted_at IS NULL", courseIDs, content.TypeAssignment).
			Order("modules.position ASC, contents.position ASC").
			Find(&allContents).Error
		if err != nil {
			return nil, err
		}
		for _, cnt := range allContents {
			if cnt.Module != nil {
				contentsByCourse[cnt.Module.CourseID] = append(contentsByCourse[cnt.Module.CourseID], cnt)
			}
		}
	}

	lookup, err := s.buildCourseScoreLookup([]uint{u.ID}, allContents, false)
	if err != nil {
		return nil, err
	}

	dtos := make([]CourseFinalGradeDto, 0, len(enrollments))
	for _, enr := range enrollments {
		crs := enr.Course
		if crs == nil {
			continue
		}

		earned, totalMax := computeStudentCourseTotal(u.ID, contentsByCourse[crs.ID], lookup, true)
		pct := 0.0
		if totalMax > 0 {
			pct = roundPercent2((earned / totalMax) * 100.0)
		}
		finalGrade := crs.CalculateGrade(pct)

		dtos = append(dtos, CourseFinalGradeDto{
			CourseUuid:    crs.Uuid,
			CourseName:    crs.Title,
			CourseCode:    fmt.Sprintf("%s %d", crs.Term, crs.Year),
			TotalScore:    earned,
			TotalMaxScore: totalMax,
			Percent:       pct,
			FinalGrade:    finalGrade,
			GradingScale:  course.GradingScaleDto{}.FromModel(crs.GradingScale),
		})
	}

	return dtos, nil
}

// GetMyCourseGradeOverview returns every counted hand-in and quiz in a course with the caller's status on each,
// plus course totals that match GetStudentFinalGrades. Unreleased grades are reported as AWAITING_RELEASE without a score.
func (s *GradeService) GetMyCourseGradeOverview(userUuid, courseUuid uuid.UUID) (*CourseGradeOverviewDto, error) {
	u, err := s.findUserByUUID(userUuid)
	if err != nil {
		return nil, err
	}

	var crs course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&crs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, course.ErrCourseNotFound
		}
		return nil, err
	}

	var contents []content.Content
	err = s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Preload("Module").
		Preload("Assignment").
		Preload("Assignment.Quiz").
		Where("modules.course_id = ? AND contents.type = ? AND contents.deleted_at IS NULL", crs.ID, content.TypeAssignment).
		Order("modules.position ASC, contents.position ASC").
		Find(&contents).Error
	if err != nil {
		return nil, err
	}

	lookup, err := s.buildCourseScoreLookup([]uint{u.ID}, contents, false)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]CourseGradeItemDto, 0, len(contents))
	gradedMax := 0.0
	for _, cnt := range contents {
		if !isCountedAssignment(cnt) {
			continue
		}
		// Students must not see items that are not open yet.
		if cnt.VisibleFrom != nil && now.Before(*cnt.VisibleFrom) {
			continue
		}

		var item CourseGradeItemDto
		key := userContentKey{userID: u.ID, contentID: cnt.ID}
		if cnt.Assignment.Type == content.AssignmentQuiz {
			item = buildQuizGradeItem(cnt, lookup, key)
		} else {
			item = buildHandinGradeItem(cnt, lookup, key)
		}
		if cnt.Module != nil {
			item.ModuleUuid = cnt.Module.Uuid
			item.ModuleTitle = cnt.Module.Title
		}
		if item.Status == CourseGradeStatusGraded {
			gradedMax += item.MaxPoints
		}
		items = append(items, item)
	}

	earned, totalMax := computeStudentCourseTotal(u.ID, contents, lookup, true)
	pct := 0.0
	if totalMax > 0 {
		pct = roundPercent2((earned / totalMax) * 100.0)
	}
	gradedPct := 0.0
	if gradedMax > 0 {
		gradedPct = roundPercent2((earned / gradedMax) * 100.0)
	}

	return &CourseGradeOverviewDto{
		CourseUuid:    crs.Uuid,
		CourseName:    crs.Title,
		CourseCode:    fmt.Sprintf("%s %d", crs.Term, crs.Year),
		TotalScore:    earned,
		TotalMaxScore: totalMax,
		Percent:       pct,
		GradedPercent: gradedPct,
		FinalGrade:    crs.CalculateGrade(pct),
		GradingScale:  course.GradingScaleDto{}.FromModel(crs.GradingScale),
		Items:         items,
	}, nil
}

func buildQuizGradeItem(cnt content.Content, lookup *courseScoreLookup, key userContentKey) CourseGradeItemDto {
	item := CourseGradeItemDto{
		Type:        GradeTypeQuiz,
		ContentUuid: cnt.Uuid,
		Title:       cnt.Title,
		Deadline:    cnt.Assignment.Deadline,
		Status:      CourseGradeStatusNotSubmitted,
		MaxPoints:   100,
		Attempt:     lookup.quizAttemptCount[key],
	}
	if cnt.Assignment.Quiz != nil {
		item.MaxAttempts = cnt.Assignment.Quiz.MaxAttempts
	}
	best, ok := lookup.quizBest[key]
	if !ok {
		return item
	}
	percent := best.Percent
	passed := best.Passed
	item.Status = CourseGradeStatusGraded
	item.Score = &percent
	item.Percent = &percent
	item.Passed = &passed
	item.GradedAt = best.SubmittedAt
	return item
}

func buildHandinGradeItem(cnt content.Content, lookup *courseScoreLookup, key userContentKey) CourseGradeItemDto {
	asgn := cnt.Assignment
	maxPts := asgn.MaxPoints
	if maxPts <= 0 {
		maxPts = content.DefaultHandinMaxPoints
	}
	item := CourseGradeItemDto{
		Type:        GradeTypeHandin,
		ContentUuid: cnt.Uuid,
		Title:       cnt.Title,
		Deadline:    asgn.Deadline,
		Status:      CourseGradeStatusNotSubmitted,
		MaxPoints:   maxPts,
		IsGroup:     asgn.AllowsGroupSubmission,
	}

	entries := lookup.handinSubs[key]
	if asgn.AllowsGroupSubmission {
		entries = lookup.groupHandinSubs[key]
	}
	if len(entries) == 0 {
		return item
	}

	// Entries are ordered newest attempt first. Same rule as computeStudentCourseTotal: first visible grade wins.
	for _, entry := range entries {
		if entry.grade == nil || !entry.grade.IsVisibleToStudent {
			continue
		}
		score := entry.grade.Score
		percent := roundPercent2((score / maxPts) * 100.0)
		gradedAt := entry.grade.UpdatedAt
		item.Status = CourseGradeStatusGraded
		item.Score = &score
		item.Percent = &percent
		item.Attempt = entry.sub.Attempt
		item.IsLate = entry.sub.IsLate
		item.GroupName = entry.groupName
		item.Feedback = entry.grade.Feedback
		item.GradedAt = &gradedAt
		return item
	}

	latest := entries[0]
	item.Attempt = latest.sub.Attempt
	item.IsLate = latest.sub.IsLate
	item.GroupName = latest.groupName
	item.Status = CourseGradeStatusSubmitted
	if latest.grade != nil {
		item.Status = CourseGradeStatusAwaitingRelease
	}
	return item
}

// GetCourseRosterFinalGrades returns all enrolled students in a course with their cumulative scores, percentages, and final letter grades.
func (s *GradeService) GetCourseRosterFinalGrades(courseUuid uuid.UUID) ([]StudentFinalGradeRowDto, error) {
	var crs course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&crs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, course.ErrCourseNotFound
		}
		return nil, err
	}

	var enrollments []course.Enrollment
	err := s.db.Preload("User").
		Where("course_id = ? AND role = ? AND deleted_at IS NULL", crs.ID, course.RoleStudent).
		Order("user_email ASC").
		Find(&enrollments).Error
	if err != nil {
		return nil, err
	}

	var contents []content.Content
	err = s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Preload("Assignment").
		Preload("Assignment.Quiz").
		Where("modules.course_id = ? AND contents.type = ? AND contents.deleted_at IS NULL", crs.ID, content.TypeAssignment).
		Order("modules.position ASC, contents.position ASC").
		Find(&contents).Error
	if err != nil {
		return nil, err
	}

	studentIDs := make([]uint, 0, len(enrollments))
	for _, enr := range enrollments {
		if enr.User != nil {
			studentIDs = append(studentIDs, enr.User.ID)
		}
	}

	lookup, err := s.buildCourseScoreLookup(studentIDs, contents, false)
	if err != nil {
		return nil, err
	}

	dtos := make([]StudentFinalGradeRowDto, 0, len(enrollments))
	for _, enr := range enrollments {
		stUser := enr.User
		if stUser == nil {
			continue
		}

		earned, totalMax := computeStudentCourseTotal(stUser.ID, contents, lookup, false)
		pct := 0.0
		if totalMax > 0 {
			pct = roundPercent2((earned / totalMax) * 100.0)
		}
		finalGrade := crs.CalculateGrade(pct)

		dtos = append(dtos, StudentFinalGradeRowDto{
			UserUuid:      stUser.Uuid,
			UserEmail:     enr.UserEmail,
			UserName:      stUser.Username,
			TotalScore:    earned,
			TotalMaxScore: totalMax,
			Percent:       pct,
			FinalGrade:    finalGrade,
		})
	}

	return dtos, nil
}
