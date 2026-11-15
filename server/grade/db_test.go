package grade

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type mockNotifier struct {
	mu           sync.Mutex
	singleCalls  []notification.NotificationContent
	manyCalls    [][]uuid.UUID
	singleRecips []uuid.UUID
}

func (m *mockNotifier) NotifySystem(recipientUuid uuid.UUID, content notification.NotificationContent) (*notification.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.singleCalls = append(m.singleCalls, content)
	m.singleRecips = append(m.singleRecips, recipientUuid)
	return &notification.Notification{Uuid: uuid.New(), UserID: 1}, nil
}

func (m *mockNotifier) NotifySystemMany(recipientUuids []uuid.UUID, content notification.NotificationContent) ([]notification.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.manyCalls = append(m.manyCalls, recipientUuids)
	notifs := make([]notification.Notification, len(recipientUuids))
	return notifs, nil
}

func setupGradeTestDb(t *testing.T) (*gorm.DB, *mockNotifier, IGradeService) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Module{},
		&course.Enrollment{},
		&course.Group{},
		&content.Content{},
		&content.Assignment{},
		&content.Quiz{},
		&quiz.Attempt{},
		&submission.IndividualSubmission{},
		&submission.GroupSubmission{},
		&submission.Grade{},
		&notification.Notification{},
	)

	notifier := &mockNotifier{}
	logger := zap.NewNop().Sugar()
	svc := NewGradeService(db, notifier, logger)
	return db, notifier, svc
}

func TestGradeService_UpsertGrade(t *testing.T) {
	db, notifier, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher", Email: "teacher@example.com"}
	student := user.User{Uuid: uuid.New(), Username: "student", Email: "student@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&student).Error)

	crs := course.Course{Uuid: uuid.New(), Title: "DAT250", Term: "AUTUMN", Year: 2026}
	assert.NoError(t, db.Create(&crs).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	cnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Assignment 1", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&cnt).Error)

	asgn := content.Assignment{ContentID: cnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100.0}
	assert.NoError(t, db.Create(&asgn).Error)

	sub := submission.IndividualSubmission{
		Submission: submission.Submission{
			Uuid:        uuid.New(),
			ContentID:   cnt.ID,
			Attempt:     1,
			SubmittedAt: time.Now(),
		},
		UserID:    &student.ID,
		UserEmail: student.Email,
	}
	assert.NoError(t, db.Create(&sub).Error)

	t.Run("creates draft grade (not visible, no notification)", func(t *testing.T) {
		score := 85.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Good start, check edge cases.",
			IsVisibleToStudent: false,
		}

		res, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 85.0, res.Score)
		assert.Equal(t, 100.0, res.MaxPoints)
		assert.Equal(t, "Good start, check edge cases.", res.Feedback)
		assert.False(t, res.IsVisibleToStudent)
		assert.Nil(t, res.ReleasedAt)
		assert.Equal(t, teacher.Email, res.GradedByEmail)
		assert.Empty(t, notifier.singleCalls)
	})

	t.Run("updates existing grade and releases it (notifies student)", func(t *testing.T) {
		score := 90.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Excellent updates!",
			IsVisibleToStudent: true,
		}

		res, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 90.0, res.Score)
		assert.True(t, res.IsVisibleToStudent)
		assert.NotNil(t, res.ReleasedAt)
		assert.Len(t, notifier.singleCalls, 1)
		assert.Equal(t, student.Uuid, notifier.singleRecips[0])
		assert.Contains(t, notifier.singleCalls[0].Title, "Assignment 1")
	})

	t.Run("updating already visible grade does not notify again", func(t *testing.T) {
		score := 95.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Bumped for outstanding test suite.",
			IsVisibleToStudent: true,
		}

		res, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.NoError(t, err)
		assert.Equal(t, 95.0, res.Score)
		// Calls count should still be 1
		assert.Len(t, notifier.singleCalls, 1)
	})

	t.Run("un-releasing grade sets ReleasedAt to nil", func(t *testing.T) {
		score := 95.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Temporarily hiding",
			IsVisibleToStudent: false,
		}

		res, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.NoError(t, err)
		assert.False(t, res.IsVisibleToStudent)
		assert.Nil(t, res.ReleasedAt)
	})

	t.Run("target is a group submission returns ErrGroupGradingNotSupported", func(t *testing.T) {
		grpSub := submission.GroupSubmission{
			Submission: submission.Submission{
				Uuid:        uuid.New(),
				ContentID:   cnt.ID,
				Attempt:     1,
				SubmittedAt: time.Now(),
			},
		}
		assert.NoError(t, db.Create(&grpSub).Error)

		score := 80.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, grpSub.Uuid, teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrGroupGradingNotSupported)
	})

	t.Run("creates grade directly as visible (notifies immediately)", func(t *testing.T) {
		subNew := submission.IndividualSubmission{
			Submission: submission.Submission{
				Uuid:        uuid.New(),
				ContentID:   cnt.ID,
				Attempt:     2,
				SubmittedAt: time.Now(),
			},
			UserID:    &student.ID,
			UserEmail: student.Email,
		}
		assert.NoError(t, db.Create(&subNew).Error)

		score := 99.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Direct release",
			IsVisibleToStudent: true,
		}
		res, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, subNew.Uuid, teacher.Uuid, dto)
		assert.NoError(t, err)
		assert.True(t, res.IsVisibleToStudent)
		assert.NotNil(t, res.ReleasedAt)
	})

	t.Run("score exceeds maxPoints returns ErrScoreOutOfRange", func(t *testing.T) {
		score := 150.0
		dto := UpsertGradeDto{
			Score:              &score,
			Feedback:           "Too high",
			IsVisibleToStudent: true,
		}

		_, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrScoreOutOfRange)
	})

	t.Run("grader not found returns ErrGraderNotFound", func(t *testing.T) {
		score := 50.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub.Uuid, uuid.New(), dto)
		assert.ErrorIs(t, err, ErrGraderNotFound)
	})

	t.Run("calling release on assignment with no grades returns 0", func(t *testing.T) {
		emptyCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "HW Empty", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&emptyCnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: emptyCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}).Error)

		rel, err := svc.ReleaseAssignmentGrades(crs.Uuid, emptyCnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 0, rel)
	})

	t.Run("content not found returns ErrContentNotFound", func(t *testing.T) {
		score := 50.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, uuid.New(), sub.Uuid, teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrContentNotFound)
	})

	t.Run("submission not found returns ErrSubmissionNotFound", func(t *testing.T) {
		score := 50.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, uuid.New(), teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrSubmissionNotFound)
	})

	t.Run("non-handin returns ErrNotHandin", func(t *testing.T) {
		quizCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Quiz 1", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&quizCnt).Error)
		quizAsgn := content.Assignment{ContentID: quizCnt.ID, Type: content.AssignmentQuiz}
		assert.NoError(t, db.Create(&quizAsgn).Error)

		score := 50.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, quizCnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrNotHandin)
	})

	t.Run("group assignment returns ErrGroupGradingNotSupported", func(t *testing.T) {
		grpCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Group Assignment", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&grpCnt).Error)
		grpAsgn := content.Assignment{ContentID: grpCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true}
		assert.NoError(t, db.Create(&grpAsgn).Error)

		score := 50.0
		dto := UpsertGradeDto{Score: &score}
		_, err := svc.UpsertGrade(crs.Uuid, grpCnt.Uuid, sub.Uuid, teacher.Uuid, dto)
		assert.ErrorIs(t, err, ErrGroupGradingNotSupported)
	})
}

func TestGradeService_ReleaseAssignmentGrades(t *testing.T) {
	db, notifier, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher", Email: "teacher@example.com"}
	st1 := user.User{Uuid: uuid.New(), Username: "st1", Email: "st1@example.com"}
	st2 := user.User{Uuid: uuid.New(), Username: "st2", Email: "st2@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&st1).Error)
	assert.NoError(t, db.Create(&st2).Error)

	crs := course.Course{Uuid: uuid.New(), Title: "Course 1"}
	assert.NoError(t, db.Create(&crs).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "M1"}
	assert.NoError(t, db.Create(&mod).Error)

	cnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "HW 1", Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&cnt).Error)

	asgn := content.Assignment{ContentID: cnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}
	assert.NoError(t, db.Create(&asgn).Error)

	sub1 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st1.ID,
		UserEmail:  st1.Email,
	}
	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st2.ID,
		UserEmail:  st2.Email,
	}
	assert.NoError(t, db.Create(&sub1).Error)
	assert.NoError(t, db.Create(&sub2).Error)

	g1 := submission.Grade{IndividualSubmissionID: &sub1.ID, Score: 80, IsVisibleToStudent: false}
	g2 := submission.Grade{IndividualSubmissionID: &sub2.ID, Score: 90, IsVisibleToStudent: false}
	assert.NoError(t, db.Create(&g1).Error)
	assert.NoError(t, db.Create(&g2).Error)

	t.Run("releases all unreleased grades and notifies students", func(t *testing.T) {
		released, err := svc.ReleaseAssignmentGrades(crs.Uuid, cnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 2, released)
		assert.Len(t, notifier.manyCalls, 1)
		assert.Len(t, notifier.manyCalls[0], 2)

		// Second call returns 0
		released2, err := svc.ReleaseAssignmentGrades(crs.Uuid, cnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 0, released2)
	})

	t.Run("calling release on assignment with no grades returns 0", func(t *testing.T) {
		emptyCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "HW Empty", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&emptyCnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: emptyCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}).Error)

		rel, err := svc.ReleaseAssignmentGrades(crs.Uuid, emptyCnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 0, rel)
	})

	t.Run("content not found returns ErrContentNotFound", func(t *testing.T) {
		_, err := svc.ReleaseAssignmentGrades(crs.Uuid, uuid.New())
		assert.ErrorIs(t, err, ErrContentNotFound)
	})

	t.Run("non-handin returns ErrNotHandin", func(t *testing.T) {
		quizCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Q", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&quizCnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: quizCnt.ID, Type: content.AssignmentQuiz}).Error)

		_, err := svc.ReleaseAssignmentGrades(crs.Uuid, quizCnt.Uuid)
		assert.ErrorIs(t, err, ErrNotHandin)
	})

	t.Run("group assignment returns ErrGroupGradingNotSupported", func(t *testing.T) {
		grpCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "G", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&grpCnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: grpCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true}).Error)

		_, err := svc.ReleaseAssignmentGrades(crs.Uuid, grpCnt.Uuid)
		assert.ErrorIs(t, err, ErrGroupGradingNotSupported)
	})
}

func TestGradeService_ListPending(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher", Email: "teacher@example.com"}
	st := user.User{Uuid: uuid.New(), Username: "st", Email: "st@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&st).Error)

	crs := course.Course{Uuid: uuid.New(), Title: "Course A", Term: "SPRING", Year: 2026}
	assert.NoError(t, db.Create(&crs).Error)

	// Enroll teacher
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: teacher.ID, UserEmail: teacher.Email, Role: course.RoleTeacher, EnrolledAt: time.Now()}).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module A"}
	assert.NoError(t, db.Create(&mod).Error)

	cnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "HW 1", Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&cnt).Error)
	asgn := content.Assignment{ContentID: cnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}
	assert.NoError(t, db.Create(&asgn).Error)

	// Attempt 1 (older)
	sub1 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: 1, SubmittedAt: time.Now().Add(-1 * time.Hour)},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub1).Error)

	// Attempt 2 (latest)
	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: 2, SubmittedAt: time.Now()},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)

	t.Run("returns only latest ungraded attempt", func(t *testing.T) {
		pending, err := svc.ListPending(teacher.Uuid, false, &crs.Uuid)
		assert.NoError(t, err)
		assert.Len(t, pending, 1)
		assert.Equal(t, sub2.Uuid, pending[0].SubmissionUuid)
		assert.Equal(t, 2, pending[0].Attempt)
		assert.Equal(t, "Course A", pending[0].CourseName)
	})

	t.Run("grading latest attempt removes it from pending list", func(t *testing.T) {
		score := 90.0
		_, err := svc.UpsertGrade(crs.Uuid, cnt.Uuid, sub2.Uuid, teacher.Uuid, UpsertGradeDto{Score: &score})
		assert.NoError(t, err)

		pending, err := svc.ListPending(teacher.Uuid, false, nil)
		assert.NoError(t, err)
		assert.Empty(t, pending)
	})

	t.Run("admin sees pending submissions without enrollment", func(t *testing.T) {
		adminUser := user.User{Uuid: uuid.New(), Username: "admin", Email: "admin@example.com"}
		assert.NoError(t, db.Create(&adminUser).Error)

		// Create another ungraded submission for st
		sub3 := submission.IndividualSubmission{
			Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: 3, SubmittedAt: time.Now()},
			UserID:     &st.ID,
			UserEmail:  st.Email,
		}
		assert.NoError(t, db.Create(&sub3).Error)

		pending, err := svc.ListPending(adminUser.Uuid, true, &crs.Uuid)
		assert.NoError(t, err)
		assert.NotEmpty(t, pending)
	})

	t.Run("non-existent user returns error in ListPending", func(t *testing.T) {
		_, err := svc.ListPending(uuid.New(), false, nil)
		assert.Error(t, err)
	})

	t.Run("non-teacher sees no pending submissions", func(t *testing.T) {
		pending, err := svc.ListPending(st.Uuid, false, nil)
		assert.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestGradeService_ListMyGrades(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher", Email: "teacher@example.com"}
	st := user.User{Uuid: uuid.New(), Username: "student", Email: "student@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&st).Error)

	crs := course.Course{Uuid: uuid.New(), Title: "Course 101", Term: "AUTUMN", Year: 2026}
	assert.NoError(t, db.Create(&crs).Error)

	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st.ID, UserEmail: st.Email, Role: course.RoleStudent, EnrolledAt: time.Now()}).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	// 1. Handin assignment with visible grade
	hCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Lab 1", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&hCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: hCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}).Error)

	sub := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub).Error)

	relAt := time.Now()
	grd := submission.Grade{
		IndividualSubmissionID: &sub.ID,
		Score:                  92.0,
		Feedback:               "Great work!",
		IsVisibleToStudent:     true,
		ReleasedAt:             &relAt,
		GradedByID:             &teacher.ID,
	}
	assert.NoError(t, db.Create(&grd).Error)

	// 2. Published quiz with 2 attempts (best is 80%)
	qCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Quiz 1", Position: 2, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&qCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: qCnt.ID, Type: content.AssignmentQuiz}).Error)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: qCnt.ID, Published: true}).Error)

	now := time.Now()
	att1 := quiz.Attempt{UserID: st.ID, QuizID: qCnt.ID, StartedAt: now.Add(-1 * time.Hour), SubmittedAt: &now, Percent: 60.0}
	att2 := quiz.Attempt{UserID: st.ID, QuizID: qCnt.ID, StartedAt: now, SubmittedAt: &now, Percent: 80.0}
	assert.NoError(t, db.Create(&att1).Error)
	assert.NoError(t, db.Create(&att2).Error)

	t.Run("returns handin grade and quiz best score", func(t *testing.T) {
		grades, err := svc.ListMyGrades(st.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, grades, 2)

		assert.Equal(t, GradeTypeHandin, grades[0].Type)
		assert.Equal(t, "Lab 1", grades[0].AssignmentTitle)
		assert.Equal(t, 92.0, grades[0].Score)
		assert.Equal(t, 92.0, grades[0].Percent)
		assert.Equal(t, "Great work!", grades[0].Feedback)

		assert.Equal(t, GradeTypeQuiz, grades[1].Type)
		assert.Equal(t, "Quiz 1", grades[1].AssignmentTitle)
		assert.Equal(t, 80.0, grades[1].Score)
		assert.Equal(t, 80.0, grades[1].Percent)
	})

	t.Run("filter by courseUuid returns only matching course grades", func(t *testing.T) {
		grades, err := svc.ListMyGrades(st.Uuid, &crs.Uuid)
		assert.NoError(t, err)
		assert.NotEmpty(t, grades)

		randomUuid := uuid.New()
		emptyGrades, err := svc.ListMyGrades(st.Uuid, &randomUuid)
		assert.NoError(t, err)
		assert.Empty(t, emptyGrades)
	})

	t.Run("non-existent student returns error", func(t *testing.T) {
		_, err := svc.ListMyGrades(uuid.New(), nil)
		assert.Error(t, err)
	})

	t.Run("hiding grade excludes it from student view", func(t *testing.T) {
		assert.NoError(t, db.Model(&grd).Update("is_visible_to_student", false).Error)

		grades, err := svc.ListMyGrades(st.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, grades, 1)
		assert.Equal(t, GradeTypeQuiz, grades[0].Type)
	})
}

func TestGradeService_ExportCourseCSV(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher", Email: "teacher@example.com"}
	st := user.User{Uuid: uuid.New(), Username: "student", Email: "student@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&st).Error)

	crs := course.Course{Uuid: uuid.New(), Title: "Course Export"}
	assert.NoError(t, db.Create(&crs).Error)

	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st.ID, UserEmail: st.Email, Role: course.RoleStudent, EnrolledAt: time.Now()}).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	// Individual Handin
	hCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Project", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&hCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: hCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}).Error)

	sub := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub.ID, Score: 88, IsVisibleToStudent: true, GradedByID: &teacher.ID}).Error)

	// Second student with late ungraded submission
	st2 := user.User{Uuid: uuid.New(), Username: "student2", Email: "student2@example.com"}
	assert.NoError(t, db.Create(&st2).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st2.ID, UserEmail: st2.Email, Role: course.RoleStudent, EnrolledAt: time.Now()}).Error)

	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, IsLate: true, SubmittedAt: time.Now()},
		UserID:     &st2.ID,
		UserEmail:  st2.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub2.ID, Score: 70, IsVisibleToStudent: false, GradedByID: &teacher.ID}).Error)

	// Quiz content with no attempts for st2
	qCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Quiz Export", Position: 3, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&qCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: qCnt.ID, Type: content.AssignmentQuiz}).Error)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: qCnt.ID, Published: true}).Error)

	// Group Handin
	gCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Group Work", Position: 2, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&gCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: gCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true, MaxPoints: 100}).Error)

	// File content without assignment
	fileCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Syllabus File", Position: 4, Type: content.TypeFile}
	assert.NoError(t, db.Create(&fileCnt).Error)

	t.Run("exports CSV with individual grades and N/A for group assignments", func(t *testing.T) {
		var buf bytes.Buffer
		err := svc.ExportCourseCSV(crs.Uuid, &buf)
		assert.NoError(t, err)

		out := buf.String()
		assert.True(t, strings.Contains(out, "student@example.com"))
		assert.True(t, strings.Contains(out, "Project"))
		assert.True(t, strings.Contains(out, "88.00"))
		assert.True(t, strings.Contains(out, "Group Work"))
		assert.True(t, strings.Contains(out, "N/A (Group)"))
	})

	t.Run("non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		var buf bytes.Buffer
		err := svc.ExportCourseCSV(uuid.New(), &buf)
		assert.ErrorIs(t, err, course.ErrCourseNotFound)
	})
}
