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

	t.Run("non-existent user returns ErrUserNotFound in ListPending", func(t *testing.T) {
		_, err := svc.ListPending(uuid.New(), false, nil)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("non-teacher receives ErrNotTeacher", func(t *testing.T) {
		_, err := svc.ListPending(st.Uuid, false, nil)
		assert.ErrorIs(t, err, ErrNotTeacher)
	})

	t.Run("teacher filtering untaught or missing course receives ErrNotTeacher or ErrCourseNotFound", func(t *testing.T) {
		otherCourse := course.Course{Uuid: uuid.New(), Title: "Other", Term: "AUTUMN", Year: 2026}
		assert.NoError(t, db.Create(&otherCourse).Error)

		_, err := svc.ListPending(teacher.Uuid, false, &otherCourse.Uuid)
		assert.ErrorIs(t, err, ErrNotTeacher)

		missingUuid := uuid.New()
		_, err = svc.ListPending(teacher.Uuid, false, &missingUuid)
		assert.ErrorIs(t, err, course.ErrCourseNotFound)
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

	t.Run("non-existent student returns ErrUserNotFound", func(t *testing.T) {
		_, err := svc.ListMyGrades(uuid.New(), nil)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("returns only latest released handin grade per assignment", func(t *testing.T) {
		// Seed a second attempt whose grade is initially hidden, then released.
		sub2 := submission.IndividualSubmission{
			Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 2, SubmittedAt: time.Now().Add(time.Minute)},
			UserID:     &st.ID,
			UserEmail:  st.Email,
		}
		assert.NoError(t, db.Create(&sub2).Error)
		grd2 := submission.Grade{
			IndividualSubmissionID: &sub2.ID,
			Score:                  99.0,
			IsVisibleToStudent:     false,
			GradedByID:             &teacher.ID,
		}
		assert.NoError(t, db.Create(&grd2).Error)

		// While attempt 2 is hidden, attempt 1 (released, 92.0) is still shown.
		grades, err := svc.ListMyGrades(st.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, grades, 2)
		assert.Equal(t, 1, grades[0].Attempt)
		assert.Equal(t, 92.0, grades[0].Score)

		// Once attempt 2 is released, only attempt 2 (99.0) is returned for that hand-in.
		assert.NoError(t, db.Model(&grd2).Update("is_visible_to_student", true).Error)
		grades, err = svc.ListMyGrades(st.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, grades, 2)
		assert.Equal(t, 2, grades[0].Attempt)
		assert.Equal(t, 99.0, grades[0].Score)
	})

	t.Run("hiding grade excludes it from student view", func(t *testing.T) {
		assert.NoError(t, db.Model(&submission.Grade{}).Where("1 = 1").Update("is_visible_to_student", false).Error)

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

	// Unpublished draft quiz must be ignored by CSV export and max totals
	draftQCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Draft Quiz", Position: 5, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&draftQCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: draftQCnt.ID, Type: content.AssignmentQuiz}).Error)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: draftQCnt.ID, Published: false}).Error)

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
		assert.False(t, strings.Contains(out, "Draft Quiz"))
	})

	t.Run("non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		var buf bytes.Buffer
		err := svc.ExportCourseCSV(uuid.New(), &buf)
		assert.ErrorIs(t, err, course.ErrCourseNotFound)
	})
}

func TestGradeService_FinalGrades(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	crs := course.Course{
		Uuid:         uuid.New(),
		Title:        "Advanced Databases",
		Term:         course.TermAutumn,
		Year:         2026,
		GradingScale: course.DefaultGradingScale(),
	}
	assert.NoError(t, db.Create(&crs).Error)

	mod := course.Module{CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	st1 := user.User{Uuid: uuid.New(), Username: "student_one", Email: "st1@example.com"}
	st2 := user.User{Uuid: uuid.New(), Username: "student_two", Email: "st2@example.com"}
	assert.NoError(t, db.Create(&st1).Error)
	assert.NoError(t, db.Create(&st2).Error)

	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st1.ID, UserEmail: st1.Email, Role: course.RoleStudent}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st2.ID, UserEmail: st2.Email, Role: course.RoleStudent}).Error)

	// 1. Handin assignment (MaxPoints: 100)
	hCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Homework 1", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&hCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: hCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 100}).Error)

	// Submissions:
	// Student 1: 95.0, visible
	sub1 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st1.ID,
		UserEmail:  st1.Email,
	}
	assert.NoError(t, db.Create(&sub1).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub1.ID, Score: 95.0, IsVisibleToStudent: true}).Error)

	// Student 2: 75.0, hidden
	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st2.ID,
		UserEmail:  st2.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub2.ID, Score: 75.0, IsVisibleToStudent: false}).Error)

	// 2. Quiz assignment (100 pts)
	qCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Quiz 1", Position: 2, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&qCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: qCnt.ID, Type: content.AssignmentQuiz}).Error)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: qCnt.ID, Published: true}).Error)

	now := time.Now()
	// Student 1 has a quiz attempt with 85%
	assert.NoError(t, db.Create(&quiz.Attempt{
		UserID:      st1.ID,
		QuizID:      qCnt.ID,
		StartedAt:   now.Add(-30 * time.Minute),
		SubmittedAt: &now,
		Percent:     85.0,
		Passed:      true,
	}).Error)

	t.Run("GetStudentFinalGrades calculates cumulative points and final letter grade", func(t *testing.T) {
		// Total max points = 200 (100 handin + 100 quiz)
		// Student 1 earned = 95.0 + 85.0 = 180.0
		// Percent = 180 / 200 = 90.0% -> Grade "A"
		finals, err := svc.GetStudentFinalGrades(st1.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, finals, 1)
		assert.Equal(t, 180.0, finals[0].TotalScore)
		assert.Equal(t, 200.0, finals[0].TotalMaxScore)
		assert.Equal(t, 90.0, finals[0].Percent)
		assert.Equal(t, "A", finals[0].FinalGrade)

		// Filter by courseUuid
		finalsFiltered, err := svc.GetStudentFinalGrades(st1.Uuid, &crs.Uuid)
		assert.NoError(t, err)
		assert.Len(t, finalsFiltered, 1)
	})

	t.Run("GetStudentFinalGrades hides unreleased grades for student view", func(t *testing.T) {
		// Student 2 has hidden grade 75.0, quiz attempt 0
		// Earned = 0.0, total max = 200.0, percent = 0.0% -> Grade "F"
		finals, err := svc.GetStudentFinalGrades(st2.Uuid, nil)
		assert.NoError(t, err)
		assert.Len(t, finals, 1)
		assert.Equal(t, 0.0, finals[0].TotalScore)
		assert.Equal(t, 200.0, finals[0].TotalMaxScore)
		assert.Equal(t, 0.0, finals[0].Percent)
		assert.Equal(t, "F", finals[0].FinalGrade)
	})

	t.Run("GetCourseRosterFinalGrades returns class-wide standing for instructors", func(t *testing.T) {
		roster, err := svc.GetCourseRosterFinalGrades(crs.Uuid)
		assert.NoError(t, err)
		assert.Len(t, roster, 2)

		// Student 1
		assert.Equal(t, "st1@example.com", roster[0].UserEmail)
		assert.Equal(t, 180.0, roster[0].TotalScore)
		assert.Equal(t, 90.0, roster[0].Percent)
		assert.Equal(t, "A", roster[0].FinalGrade)

		// Student 2 (instructor view includes unreleased score 75.0)
		assert.Equal(t, "st2@example.com", roster[1].UserEmail)
		assert.Equal(t, 75.0, roster[1].TotalScore)
		assert.Equal(t, 37.5, roster[1].Percent) // 75 / 200 = 37.5% -> F (< 40%)
		assert.Equal(t, "F", roster[1].FinalGrade)
	})

	t.Run("GetCourseRosterFinalGrades returns ErrCourseNotFound for invalid course", func(t *testing.T) {
		_, err := svc.GetCourseRosterFinalGrades(uuid.New())
		assert.ErrorIs(t, err, course.ErrCourseNotFound)
	})

	t.Run("GetStudentFinalGrades returns ErrUserNotFound for non-existent user", func(t *testing.T) {
		_, err := svc.GetStudentFinalGrades(uuid.New(), nil)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestGradeService_FinalGradeEdgeCases(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	crs := course.Course{
		Uuid:         uuid.New(),
		Title:        "Edge Cases Course",
		Term:         course.TermSpring,
		Year:         2026,
		GradingScale: course.DefaultGradingScale(),
	}
	assert.NoError(t, db.Create(&crs).Error)

	mod := course.Module{CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	st := user.User{Uuid: uuid.New(), Username: "edge_student", Email: "edge@example.com"}
	assert.NoError(t, db.Create(&st).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st.ID, UserEmail: st.Email, Role: course.RoleStudent}).Error)

	// Hand-in with MaxPoints=7 where attempt 1 is released (2.8 -> 40.00% exactly after rounding)
	// and attempt 2 has an unreleased grade (5.6).
	hCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Precision Handin", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&hCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: hCnt.ID, Type: content.AssignmentHandIn, MaxPoints: 7.0}).Error)

	// Draft unpublished quiz must NOT inflate TotalMaxScore.
	draftQuiz := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Unpublished Quiz", Position: 2, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&draftQuiz).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: draftQuiz.ID, Type: content.AssignmentQuiz}).Error)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: draftQuiz.ID, Published: false}).Error)

	sub1 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now().Add(-time.Hour)},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub1).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub1.ID, Score: 2.8, IsVisibleToStudent: true}).Error)

	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 2, SubmittedAt: time.Now()},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)
	assert.NoError(t, db.Create(&submission.Grade{IndividualSubmissionID: &sub2.ID, Score: 5.6, IsVisibleToStudent: false}).Error)

	// Student view uses attempt 1 (2.8 / 7.0 = 40.00% -> E, not F from float drift), ignores draft quiz and hidden attempt 2.
	finals, err := svc.GetStudentFinalGrades(st.Uuid, nil)
	assert.NoError(t, err)
	assert.Len(t, finals, 1)
	assert.Equal(t, 2.8, finals[0].TotalScore)
	assert.Equal(t, 7.0, finals[0].TotalMaxScore)
	assert.Equal(t, 40.0, finals[0].Percent)
	assert.Equal(t, "E", finals[0].FinalGrade)

	// Roster view (instructor) uses latest attempt 2 (5.6 / 7.0 = 80.00% -> B) and excludes draft quiz.
	roster, err := svc.GetCourseRosterFinalGrades(crs.Uuid)
	assert.NoError(t, err)
	assert.Len(t, roster, 1)
	assert.Equal(t, 5.6, roster[0].TotalScore)
	assert.Equal(t, 7.0, roster[0].TotalMaxScore)
	assert.Equal(t, 80.0, roster[0].Percent)
	assert.Equal(t, "B", roster[0].FinalGrade)
}
