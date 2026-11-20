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
		&Grade{},
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

	t.Run("group submission on individual assignment returns ErrSubmissionNotFound", func(t *testing.T) {
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
		assert.ErrorIs(t, err, ErrSubmissionNotFound)
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

	t.Run("group assignment grades group submission and notifies all members", func(t *testing.T) {
		st2 := user.User{Uuid: uuid.New(), Username: "student2", Email: "student2@example.com"}
		assert.NoError(t, db.Create(&st2).Error)

		grp := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Group 1", MaxMembers: 4}
		assert.NoError(t, db.Create(&grp).Error)

		grpCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Group Assignment", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&grpCnt).Error)
		grpAsgn := content.Assignment{ContentID: grpCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true, MaxPoints: 50}
		assert.NoError(t, db.Create(&grpAsgn).Error)

		grpSub := submission.GroupSubmission{
			Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
			GroupID:    &grp.ID,
			UserID:     &student.ID,
			UserEmail:  student.Email,
			Members:    []user.User{student, st2},
		}
		assert.NoError(t, db.Create(&grpSub).Error)

		// Missing submission returns ErrSubmissionNotFound
		score := 42.5
		_, err := svc.UpsertGrade(crs.Uuid, grpCnt.Uuid, uuid.New(), teacher.Uuid, UpsertGradeDto{Score: &score})
		assert.ErrorIs(t, err, ErrSubmissionNotFound)

		// Draft grade does not notify
		draftRes, err := svc.UpsertGrade(crs.Uuid, grpCnt.Uuid, grpSub.Uuid, teacher.Uuid, UpsertGradeDto{
			Score:              &score,
			Feedback:           "Draft group review",
			IsVisibleToStudent: false,
		})
		assert.NoError(t, err)
		assert.Equal(t, 42.5, draftRes.Score)
		assert.Equal(t, 50.0, draftRes.MaxPoints)
		assert.False(t, draftRes.IsVisibleToStudent)
		assert.Empty(t, notifier.manyCalls)

		// Releasing notifies all members via NotifySystemMany
		updatedScore := 46.0
		visRes, err := svc.UpsertGrade(crs.Uuid, grpCnt.Uuid, grpSub.Uuid, teacher.Uuid, UpsertGradeDto{
			Score:              &updatedScore,
			Feedback:           "Great teamwork!",
			IsVisibleToStudent: true,
		})
		assert.NoError(t, err)
		assert.True(t, visRes.IsVisibleToStudent)
		assert.NotNil(t, visRes.ReleasedAt)
		assert.Len(t, notifier.manyCalls, 1)
		assert.ElementsMatch(t, []uuid.UUID{student.Uuid, st2.Uuid}, notifier.manyCalls[0])
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

	g1 := Grade{IndividualSubmissionID: &sub1.ID, Score: 80, IsVisibleToStudent: false}
	g2 := Grade{IndividualSubmissionID: &sub2.ID, Score: 90, IsVisibleToStudent: false}
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

	t.Run("group assignment releases all unreleased group grades and notifies all members", func(t *testing.T) {
		grpCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Group Release HW", Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&grpCnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: grpCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true, MaxPoints: 100}).Error)

		grp := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Team A", MaxMembers: 4}
		assert.NoError(t, db.Create(&grp).Error)

		grpSub := submission.GroupSubmission{
			Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
			GroupID:    &grp.ID,
			UserID:     &st1.ID,
			UserEmail:  st1.Email,
			Members:    []user.User{st1, st2},
		}
		assert.NoError(t, db.Create(&grpSub).Error)
		assert.NoError(t, db.Create(&Grade{GroupSubmissionID: &grpSub.ID, Score: 88, IsVisibleToStudent: false}).Error)

		prevManyCalls := len(notifier.manyCalls)
		rel, err := svc.ReleaseAssignmentGrades(crs.Uuid, grpCnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 1, rel)
		assert.Len(t, notifier.manyCalls, prevManyCalls+1)
		assert.ElementsMatch(t, []uuid.UUID{st1.Uuid, st2.Uuid}, notifier.manyCalls[prevManyCalls])

		// Second call returns 0
		relAgain, err := svc.ReleaseAssignmentGrades(crs.Uuid, grpCnt.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, 0, relAgain)
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
	grd := Grade{
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
		grd2 := Grade{
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
		assert.NoError(t, db.Model(&Grade{}).Where("1 = 1").Update("is_visible_to_student", false).Error)

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
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub.ID, Score: 88, IsVisibleToStudent: true, GradedByID: &teacher.ID}).Error)

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
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub2.ID, Score: 70, IsVisibleToStudent: false, GradedByID: &teacher.ID}).Error)

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

	// Group Handin with graded submission for st & st2
	gCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Group Work", Position: 2, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&gCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: gCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true, MaxPoints: 100}).Error)

	grp := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Group Export", MaxMembers: 4}
	assert.NoError(t, db.Create(&grp).Error)
	grpSub := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: gCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		GroupID:    &grp.ID,
		UserID:     &st.ID,
		UserEmail:  st.Email,
		Members:    []user.User{st, st2},
	}
	assert.NoError(t, db.Create(&grpSub).Error)
	assert.NoError(t, db.Create(&Grade{GroupSubmissionID: &grpSub.ID, Score: 92, Feedback: "Strong team output", IsVisibleToStudent: true, GradedByID: &teacher.ID}).Error)

	// File content without assignment
	fileCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Syllabus File", Position: 4, Type: content.TypeFile}
	assert.NoError(t, db.Create(&fileCnt).Error)

	t.Run("exports CSV with individual and group grades", func(t *testing.T) {
		var buf bytes.Buffer
		err := svc.ExportCourseCSV(crs.Uuid, &buf)
		assert.NoError(t, err)

		out := buf.String()
		assert.True(t, strings.Contains(out, "student@example.com"))
		assert.True(t, strings.Contains(out, "Project"))
		assert.True(t, strings.Contains(out, "88.00"))
		assert.True(t, strings.Contains(out, "Group Work"))
		assert.True(t, strings.Contains(out, "HANDIN (GROUP)"))
		assert.True(t, strings.Contains(out, "92.00"))
		assert.True(t, strings.Contains(out, "Strong team output"))
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
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub1.ID, Score: 95.0, IsVisibleToStudent: true}).Error)

	// Student 2: 75.0, hidden
	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &st2.ID,
		UserEmail:  st2.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub2.ID, Score: 75.0, IsVisibleToStudent: false}).Error)

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
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub1.ID, Score: 2.8, IsVisibleToStudent: true}).Error)

	sub2 := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: hCnt.ID, Attempt: 2, SubmittedAt: time.Now()},
		UserID:     &st.ID,
		UserEmail:  st.Email,
	}
	assert.NoError(t, db.Create(&sub2).Error)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub2.ID, Score: 5.6, IsVisibleToStudent: false}).Error)

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

func TestGradeService_GroupGradingEndToEnd(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	teacher := user.User{Uuid: uuid.New(), Username: "teacher_g", Email: "teacher_g@example.com"}
	st1 := user.User{Uuid: uuid.New(), Username: "st1_g", Email: "st1_g@example.com"}
	st2 := user.User{Uuid: uuid.New(), Username: "st2_g", Email: "st2_g@example.com"}
	st3 := user.User{Uuid: uuid.New(), Username: "st3_g", Email: "st3_g@example.com"}
	assert.NoError(t, db.Create(&teacher).Error)
	assert.NoError(t, db.Create(&st1).Error)
	assert.NoError(t, db.Create(&st2).Error)
	assert.NoError(t, db.Create(&st3).Error)

	crs := course.Course{
		Uuid:         uuid.New(),
		Title:        "Group Systems",
		Term:         course.TermAutumn,
		Year:         2026,
		GradingScale: course.DefaultGradingScale(),
	}
	assert.NoError(t, db.Create(&crs).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: teacher.ID, UserEmail: teacher.Email, Role: course.RoleTeacher}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st1.ID, UserEmail: st1.Email, Role: course.RoleStudent}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st2.ID, UserEmail: st2.Email, Role: course.RoleStudent}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: st3.ID, UserEmail: st3.Email, Role: course.RoleStudent}).Error)

	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Project Module", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	grp := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Alpha Squad", MaxMembers: 4}
	assert.NoError(t, db.Create(&grp).Error)

	grpCnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: "Team Deliverable", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&grpCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: grpCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: true, MaxPoints: 100}).Error)

	// Attempt 1 and Attempt 2 for Alpha Squad (submitted by st1, snapshotted with st1 & st2)
	sub1 := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 1, SubmittedAt: time.Now().Add(-2 * time.Hour)},
		GroupID:    &grp.ID,
		UserID:     &st1.ID,
		UserEmail:  st1.Email,
		Members:    []user.User{st1, st2},
	}
	assert.NoError(t, db.Create(&sub1).Error)

	sub2 := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 2, SubmittedAt: time.Now().Add(-1 * time.Hour)},
		GroupID:    &grp.ID,
		UserID:     &st2.ID,
		UserEmail:  st2.Email,
		Members:    []user.User{st1, st2},
	}
	assert.NoError(t, db.Create(&sub2).Error)

	// 1. ListPending returns only latest attempt (#2) with group metadata
	pending, err := svc.ListPending(teacher.Uuid, false, &crs.Uuid)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, sub2.Uuid, pending[0].SubmissionUuid)
	assert.True(t, pending[0].IsGroup)
	assert.Equal(t, "Alpha Squad", pending[0].GroupName)
	assert.Equal(t, 2, pending[0].Attempt)
	assert.ElementsMatch(t, []string{st1.Email, st2.Email}, pending[0].MemberEmails)

	// 2. Grade attempt 2 as hidden draft -> removes from pending, hidden from students, visible in roster
	draftScore := 92.0
	_, err = svc.UpsertGrade(crs.Uuid, grpCnt.Uuid, sub2.Uuid, teacher.Uuid, UpsertGradeDto{
		Score:              &draftScore,
		Feedback:           "Solid group work",
		IsVisibleToStudent: false,
	})
	assert.NoError(t, err)

	pendingAfter, err := svc.ListPending(teacher.Uuid, false, &crs.Uuid)
	assert.NoError(t, err)
	assert.Empty(t, pendingAfter)

	// Student 1 and 2 see 0 grades and 0 final score while hidden
	myGradesHidden, err := svc.ListMyGrades(st1.Uuid, &crs.Uuid)
	assert.NoError(t, err)
	assert.Empty(t, myGradesHidden)

	finalsHidden, err := svc.GetStudentFinalGrades(st2.Uuid, &crs.Uuid)
	assert.NoError(t, err)
	assert.Len(t, finalsHidden, 1)
	assert.Equal(t, 0.0, finalsHidden[0].TotalScore)
	assert.Equal(t, 100.0, finalsHidden[0].TotalMaxScore)

	// Roster already includes unreleased group score for Alpha Squad members, 0 for st3
	roster, err := svc.GetCourseRosterFinalGrades(crs.Uuid)
	assert.NoError(t, err)
	assert.Len(t, roster, 3)
	scoreByEmail := make(map[string]float64, 3)
	for _, r := range roster {
		scoreByEmail[r.UserEmail] = r.TotalScore
	}
	assert.Equal(t, 92.0, scoreByEmail[st1.Email])
	assert.Equal(t, 92.0, scoreByEmail[st2.Email])
	assert.Equal(t, 0.0, scoreByEmail[st3.Email])

	// 3. Release all grades -> both st1 and st2 see group grade in ListMyGrades and GetStudentFinalGrades
	released, err := svc.ReleaseAssignmentGrades(crs.Uuid, grpCnt.Uuid)
	assert.NoError(t, err)
	assert.Equal(t, 1, released)

	for _, memberUuid := range []uuid.UUID{st1.Uuid, st2.Uuid} {
		myGrades, err := svc.ListMyGrades(memberUuid, &crs.Uuid)
		assert.NoError(t, err)
		assert.Len(t, myGrades, 1)
		assert.True(t, myGrades[0].IsGroup)
		assert.Equal(t, "Alpha Squad", myGrades[0].GroupName)
		assert.Equal(t, 92.0, myGrades[0].Score)
		assert.Equal(t, "Solid group work", myGrades[0].Feedback)

		finals, err := svc.GetStudentFinalGrades(memberUuid, &crs.Uuid)
		assert.NoError(t, err)
		assert.Len(t, finals, 1)
		assert.Equal(t, 92.0, finals[0].TotalScore)
		assert.Equal(t, 100.0, finals[0].TotalMaxScore)
		assert.Equal(t, "A", finals[0].FinalGrade)
	}

	// Non-member st3 still sees no grade
	myGrades3, err := svc.ListMyGrades(st3.Uuid, &crs.Uuid)
	assert.NoError(t, err)
	assert.Empty(t, myGrades3)

	// 4. Add a second module/individual handin and a second group to exercise sorting & multiple pending/graded rows
	mod2 := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Individual Module", Position: 2}
	assert.NoError(t, db.Create(&mod2).Error)
	indCnt := content.Content{Uuid: uuid.New(), ModuleID: mod2.ID, Title: "Reflection", Position: 1, Type: content.TypeAssignment}
	assert.NoError(t, db.Create(&indCnt).Error)
	assert.NoError(t, db.Create(&content.Assignment{ContentID: indCnt.ID, Type: content.AssignmentHandIn, AllowsGroupSubmission: false, MaxPoints: 50}).Error)

	indSub := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: indCnt.ID, Attempt: 1, SubmittedAt: time.Now().Add(-30 * time.Minute)},
		UserID:     &st1.ID,
		UserEmail:  st1.Email,
	}
	assert.NoError(t, db.Create(&indSub).Error)

	grp2 := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Beta Squad", MaxMembers: 4}
	assert.NoError(t, db.Create(&grp2).Error)
	grp2Sub := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 1, IsLate: true, SubmittedAt: time.Now().Add(-15 * time.Minute)},
		GroupID:    &grp2.ID,
		UserID:     &st3.ID,
		UserEmail:  st3.Email,
		Members:    []user.User{st3},
	}
	assert.NoError(t, db.Create(&grp2Sub).Error)

	// ListPending returns both indSub and grp2Sub sorted by SubmittedAt ASC
	pendingBoth, err := svc.ListPending(teacher.Uuid, false, &crs.Uuid)
	assert.NoError(t, err)
	assert.Len(t, pendingBoth, 2)
	assert.Equal(t, indSub.Uuid, pendingBoth[0].SubmissionUuid)
	assert.Equal(t, grp2Sub.Uuid, pendingBoth[1].SubmissionUuid)
	assert.True(t, pendingBoth[1].IsGroup)
	assert.True(t, pendingBoth[1].IsLate)

	// Grade and release indSub -> st1 sees both group and individual grades sorted by module/content order
	indScore := 45.0
	_, err = svc.UpsertGrade(crs.Uuid, indCnt.Uuid, indSub.Uuid, teacher.Uuid, UpsertGradeDto{
		Score:              &indScore,
		Feedback:           "Nice reflection",
		IsVisibleToStudent: true,
	})
	assert.NoError(t, err)

	allMyGrades, err := svc.ListMyGrades(st1.Uuid, nil)
	assert.NoError(t, err)
	assert.Len(t, allMyGrades, 2)
	assert.True(t, allMyGrades[0].IsGroup)
	assert.False(t, allMyGrades[1].IsGroup)

	// ExportCourseCSV includes graded Alpha Squad and ungraded late Beta Squad
	var csvBuf bytes.Buffer
	assert.NoError(t, svc.ExportCourseCSV(crs.Uuid, &csvBuf))
	csvOut := csvBuf.String()
	assert.Contains(t, csvOut, "HANDIN (GROUP)")
	assert.Contains(t, csvOut, "92.00")
	assert.Contains(t, csvOut, st3.Email)
}

func TestGradeLookup_GetSubmissionGrades(t *testing.T) {
	db, _, _ := setupGradeTestDb(t)
	lookup := NewGradeLookup(db)

	emptyMap, err := lookup.GetSubmissionGrades(nil, false, true)
	assert.NoError(t, err)
	assert.Empty(t, emptyMap)

	indVisible := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: 1, Attempt: 1, SubmittedAt: time.Now()},
	}
	indHidden := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: 1, Attempt: 2, SubmittedAt: time.Now()},
	}
	grpVisible := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: 2, Attempt: 1, SubmittedAt: time.Now()},
	}
	grpHidden := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: 2, Attempt: 2, SubmittedAt: time.Now()},
	}
	assert.NoError(t, db.Create(&indVisible).Error)
	assert.NoError(t, db.Create(&indHidden).Error)
	assert.NoError(t, db.Create(&grpVisible).Error)
	assert.NoError(t, db.Create(&grpHidden).Error)

	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &indVisible.ID, Score: 85, Feedback: "Good", IsVisibleToStudent: true}).Error)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &indHidden.ID, Score: 90, Feedback: "Draft", IsVisibleToStudent: false}).Error)
	assert.NoError(t, db.Create(&Grade{GroupSubmissionID: &grpVisible.ID, Score: 88, Feedback: "Team", IsVisibleToStudent: true}).Error)
	assert.NoError(t, db.Create(&Grade{GroupSubmissionID: &grpHidden.ID, Score: 95, Feedback: "Draft team", IsVisibleToStudent: false}).Error)

	indMapVis, err := lookup.GetSubmissionGrades([]uint{indVisible.ID, indHidden.ID}, false, true)
	assert.NoError(t, err)
	assert.Len(t, indMapVis, 1)
	assert.Equal(t, 85.0, indMapVis[indVisible.ID].Score)

	indMapAll, err := lookup.GetSubmissionGrades([]uint{indVisible.ID, indHidden.ID}, false, false)
	assert.NoError(t, err)
	assert.Len(t, indMapAll, 2)
	assert.Equal(t, 90.0, indMapAll[indHidden.ID].Score)

	grpMapVis, err := lookup.GetSubmissionGrades([]uint{grpVisible.ID, grpHidden.ID}, true, true)
	assert.NoError(t, err)
	assert.Len(t, grpMapVis, 1)
	assert.Equal(t, 88.0, grpMapVis[grpVisible.ID].Score)

	grpMapAll, err := lookup.GetSubmissionGrades([]uint{grpVisible.ID, grpHidden.ID}, true, false)
	assert.NoError(t, err)
	assert.Len(t, grpMapAll, 2)
	assert.Equal(t, 95.0, grpMapAll[grpHidden.ID].Score)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	_ = sqlDB.Close()

	_, err = lookup.GetSubmissionGrades([]uint{indVisible.ID}, false, true)
	assert.Error(t, err)
}

func TestGradeService_GetMyCourseGradeOverview(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	crs := course.Course{Uuid: uuid.New(), Title: "Overview Course", Term: course.TermAutumn, Year: 2026, GradingScale: course.DefaultGradingScale()}
	assert.NoError(t, db.Create(&crs).Error)
	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)

	me := user.User{Uuid: uuid.New(), Username: "me", Email: "me@example.com"}
	other := user.User{Uuid: uuid.New(), Username: "other", Email: "other@example.com"}
	assert.NoError(t, db.Create(&me).Error)
	assert.NoError(t, db.Create(&other).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: me.ID, UserEmail: me.Email, Role: course.RoleStudent}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: other.ID, UserEmail: other.Email, Role: course.RoleStudent}).Error)

	pos := 0
	newAssignment := func(title string, asgn content.Assignment, visibleFrom *time.Time) content.Content {
		pos++
		cnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: title, Position: pos, Type: content.TypeAssignment, VisibleFrom: visibleFrom}
		assert.NoError(t, db.Create(&cnt).Error)
		asgn.ContentID = cnt.ID
		assert.NoError(t, db.Create(&asgn).Error)
		return cnt
	}
	newSub := func(cnt content.Content, u user.User, attempt int, late bool) submission.IndividualSubmission {
		sub := submission.IndividualSubmission{
			Submission: submission.Submission{Uuid: uuid.New(), ContentID: cnt.ID, Attempt: attempt, IsLate: late, SubmittedAt: time.Now()},
			UserID:     &u.ID,
			UserEmail:  u.Email,
		}
		assert.NoError(t, db.Create(&sub).Error)
		return sub
	}
	handin := content.Assignment{Type: content.AssignmentHandIn, MaxPoints: 5}

	deadline := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	notSubmitted := newAssignment("Not submitted", content.Assignment{Type: content.AssignmentHandIn, MaxPoints: 5, Deadline: &deadline}, nil)
	submitted := newAssignment("Submitted", handin, nil)
	unreleased := newAssignment("Unreleased", handin, nil)
	released := newAssignment("Released", handin, nil)

	newSub(submitted, me, 1, false)
	unrelSub := newSub(unreleased, me, 1, false)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &unrelSub.ID, Score: 4, Feedback: "secret", IsVisibleToStudent: false}).Error)
	relSub := newSub(released, me, 1, true)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &relSub.ID, Score: 3, Feedback: "nice", IsVisibleToStudent: true}).Error)

	// Another student's released grade on the same assignment must never leak.
	otherSub := newSub(notSubmitted, other, 1, false)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &otherSub.ID, Score: 5, IsVisibleToStudent: true}).Error)

	// Group hand-in: caller is a member but not the submitter.
	grp := course.Group{Uuid: uuid.New(), CourseID: crs.ID, Name: "Alpha", MaxMembers: 4}
	assert.NoError(t, db.Create(&grp).Error)
	grpCnt := newAssignment("Group work", content.Assignment{Type: content.AssignmentHandIn, MaxPoints: 10, AllowsGroupSubmission: true}, nil)
	grpSub := submission.GroupSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: grpCnt.ID, Attempt: 1, SubmittedAt: time.Now()},
		GroupID:    &grp.ID,
		UserID:     &other.ID,
		UserEmail:  other.Email,
		Members:    []user.User{me, other},
	}
	assert.NoError(t, db.Create(&grpSub).Error)
	assert.NoError(t, db.Create(&Grade{GroupSubmissionID: &grpSub.ID, Score: 8, IsVisibleToStudent: true}).Error)

	// Quizzes: attempted (best of two), unattempted, unpublished.
	attempted := newAssignment("Quiz attempted", content.Assignment{Type: content.AssignmentQuiz}, nil)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: attempted.ID, Published: true, MaxAttempts: 3}).Error)
	unattempted := newAssignment("Quiz unattempted", content.Assignment{Type: content.AssignmentQuiz}, nil)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: unattempted.ID, Published: true, MaxAttempts: 1}).Error)
	draftQuiz := newAssignment("Quiz draft", content.Assignment{Type: content.AssignmentQuiz}, nil)
	assert.NoError(t, db.Create(&content.Quiz{ContentID: draftQuiz.ID, Published: false}).Error)

	now := time.Now()
	for _, pct := range []float64{60, 80} {
		assert.NoError(t, db.Create(&quiz.Attempt{UserID: me.ID, QuizID: attempted.ID, StartedAt: now.Add(-time.Hour), SubmittedAt: &now, Percent: pct, Passed: pct >= 50}).Error)
	}
	// In-progress attempt must not count.
	assert.NoError(t, db.Create(&quiz.Attempt{UserID: me.ID, QuizID: unattempted.ID, StartedAt: now}).Error)

	// Not open yet: hidden from the item list.
	future := time.Now().Add(48 * time.Hour)
	newAssignment("Future", handin, &future)

	overview, err := svc.GetMyCourseGradeOverview(me.Uuid, crs.Uuid)
	assert.NoError(t, err)
	if !assert.NotNil(t, overview) {
		return
	}

	byTitle := make(map[string]CourseGradeItemDto, len(overview.Items))
	titles := make([]string, 0, len(overview.Items))
	for _, it := range overview.Items {
		byTitle[it.Title] = it
		titles = append(titles, it.Title)
	}

	t.Run("lists counted visible items in module and content order", func(t *testing.T) {
		assert.Equal(t, []string{"Not submitted", "Submitted", "Unreleased", "Released", "Group work", "Quiz attempted", "Quiz unattempted"}, titles)
		assert.Equal(t, mod.Uuid, overview.Items[0].ModuleUuid)
		assert.Equal(t, "Module 1", overview.Items[0].ModuleTitle)
		assert.Equal(t, "Overview Course", overview.CourseName)
		assert.Equal(t, "AUTUMN 2026", overview.CourseCode)
	})

	t.Run("not submitted hides other student's grade", func(t *testing.T) {
		it := byTitle["Not submitted"]
		assert.Equal(t, CourseGradeStatusNotSubmitted, it.Status)
		assert.Nil(t, it.Score)
		assert.Nil(t, it.Percent)
		assert.Nil(t, it.SubmittedAt)
		assert.NotNil(t, it.Deadline)
		assert.Equal(t, 5.0, it.MaxPoints)
	})

	t.Run("submitted without grade", func(t *testing.T) {
		it := byTitle["Submitted"]
		assert.Equal(t, CourseGradeStatusSubmitted, it.Status)
		assert.NotNil(t, it.SubmittedAt)
		assert.Equal(t, 1, it.Attempt)
		assert.Nil(t, it.Score)
	})

	t.Run("graded but unreleased hides score and feedback", func(t *testing.T) {
		it := byTitle["Unreleased"]
		assert.Equal(t, CourseGradeStatusAwaitingRelease, it.Status)
		assert.Nil(t, it.Score)
		assert.Nil(t, it.Percent)
		assert.Empty(t, it.Feedback)
		assert.Nil(t, it.GradedAt)
	})

	t.Run("released grade shows score, percent, feedback, late flag", func(t *testing.T) {
		it := byTitle["Released"]
		assert.Equal(t, CourseGradeStatusGraded, it.Status)
		if assert.NotNil(t, it.Score) && assert.NotNil(t, it.Percent) {
			assert.Equal(t, 3.0, *it.Score)
			assert.Equal(t, 60.0, *it.Percent)
		}
		assert.Equal(t, "nice", it.Feedback)
		assert.NotNil(t, it.SubmittedAt)
		assert.True(t, it.IsLate)
		assert.NotNil(t, it.GradedAt)
	})

	t.Run("group hand-in graded for non-submitting member", func(t *testing.T) {
		it := byTitle["Group work"]
		assert.Equal(t, CourseGradeStatusGraded, it.Status)
		assert.True(t, it.IsGroup)
		assert.Equal(t, "Alpha", it.GroupName)
		if assert.NotNil(t, it.Score) {
			assert.Equal(t, 8.0, *it.Score)
		}
	})

	t.Run("quiz uses best finished attempt", func(t *testing.T) {
		it := byTitle["Quiz attempted"]
		assert.Equal(t, CourseGradeStatusGraded, it.Status)
		assert.Equal(t, 100.0, it.MaxPoints)
		assert.Equal(t, 2, it.Attempt)
		assert.Equal(t, 3, it.MaxAttempts)
		assert.NotNil(t, it.SubmittedAt)
		if assert.NotNil(t, it.Percent) && assert.NotNil(t, it.Passed) {
			assert.Equal(t, 80.0, *it.Percent)
			assert.True(t, *it.Passed)
		}
	})

	t.Run("unattempted quiz ignores in-progress attempt", func(t *testing.T) {
		it := byTitle["Quiz unattempted"]
		assert.Equal(t, CourseGradeStatusNotSubmitted, it.Status)
		assert.Equal(t, 0, it.Attempt)
		assert.Nil(t, it.Passed)
	})

	t.Run("totals match GetStudentFinalGrades", func(t *testing.T) {
		finals, err := svc.GetStudentFinalGrades(me.Uuid, &crs.Uuid)
		assert.NoError(t, err)
		if assert.Len(t, finals, 1) {
			assert.Equal(t, finals[0].TotalScore, overview.TotalScore)
			assert.Equal(t, finals[0].TotalMaxScore, overview.TotalMaxScore)
			assert.Equal(t, finals[0].Percent, overview.Percent)
			assert.Equal(t, finals[0].FinalGrade, overview.FinalGrade)
		}
		// 3 (released) + 8 (group) + 80 (quiz); unreleased grade excluded.
		assert.Equal(t, 91.0, overview.TotalScore)
	})

	t.Run("graded percent only counts graded items", func(t *testing.T) {
		// graded max = 5 (released) + 10 (group) + 100 (quiz) = 115; 91 / 115 = 79.13%
		assert.Equal(t, 79.13, overview.GradedPercent)
		assert.Greater(t, overview.GradedPercent, overview.Percent)
	})

	t.Run("course with pending items is not finished and progress counts graded items only", func(t *testing.T) {
		assert.False(t, overview.Finished)
		// graded items: 3 + 8 + 80 = 91 of 5 + 10 + 100 = 115. Pending items have a future or no deadline.
		assert.Equal(t, 91.0, overview.ProgressScore)
		assert.Equal(t, 115.0, overview.ProgressMaxScore)
	})

	t.Run("course with no assignments returns empty items and zero totals", func(t *testing.T) {
		empty := course.Course{Uuid: uuid.New(), Title: "Empty", Term: course.TermAutumn, Year: 2026, GradingScale: course.DefaultGradingScale()}
		assert.NoError(t, db.Create(&empty).Error)
		res, err := svc.GetMyCourseGradeOverview(me.Uuid, empty.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, res.Items)
		assert.Empty(t, res.Items)
		assert.Equal(t, 0.0, res.Percent)
		assert.Equal(t, 0.0, res.GradedPercent)
	})

	t.Run("unknown user returns ErrUserNotFound", func(t *testing.T) {
		_, err := svc.GetMyCourseGradeOverview(uuid.New(), crs.Uuid)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("unknown course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.GetMyCourseGradeOverview(me.Uuid, uuid.New())
		assert.ErrorIs(t, err, course.ErrCourseNotFound)
	})
}

func TestComputeCourseProgress(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	graded := func(score, maxPts float64, deadline *time.Time) CourseGradeItemDto {
		return CourseGradeItemDto{Status: CourseGradeStatusGraded, Score: &score, MaxPoints: maxPts, Deadline: deadline}
	}
	open := func(maxPts float64, deadline *time.Time) CourseGradeItemDto {
		return CourseGradeItemDto{Status: CourseGradeStatusNotSubmitted, MaxPoints: maxPts, Deadline: deadline}
	}

	tests := []struct {
		name         string
		items        []CourseGradeItemDto
		wantEarned   float64
		wantMax      float64
		wantFinished bool
	}{
		{"no items is not finished", nil, 0, 0, false},
		{"second item not due yet is left out", []CourseGradeItemDto{graded(4, 5, &past), open(5, &future)}, 4, 5, false},
		{"missed item past deadline counts as zero", []CourseGradeItemDto{graded(4, 5, &past), open(5, &past)}, 4, 10, true},
		{"graded item before its deadline still counts", []CourseGradeItemDto{graded(5, 5, &future)}, 5, 5, true},
		{"ungraded item without deadline stays pending", []CourseGradeItemDto{graded(4, 5, &past), open(5, nil)}, 4, 5, false},
		{"awaiting release past deadline counts max without score", []CourseGradeItemDto{
			{Status: CourseGradeStatusAwaitingRelease, MaxPoints: 5, Deadline: &past},
		}, 0, 5, true},
		{"submitted before deadline is still pending", []CourseGradeItemDto{
			{Status: CourseGradeStatusSubmitted, MaxPoints: 5, Deadline: &future},
		}, 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			earned, maxPts, finished := computeCourseProgress(tc.items, now)
			assert.Equal(t, tc.wantEarned, earned)
			assert.Equal(t, tc.wantMax, maxPts)
			assert.Equal(t, tc.wantFinished, finished)
		})
	}
}

func TestGradeService_CourseFinishedState(t *testing.T) {
	db, _, svc := setupGradeTestDb(t)

	crs := course.Course{Uuid: uuid.New(), Title: "Finish Course", Term: course.TermAutumn, Year: 2026, GradingScale: course.DefaultGradingScale()}
	assert.NoError(t, db.Create(&crs).Error)
	mod := course.Module{Uuid: uuid.New(), CourseID: crs.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(&mod).Error)
	me := user.User{Uuid: uuid.New(), Username: "finisher", Email: "finisher@example.com"}
	assert.NoError(t, db.Create(&me).Error)
	assert.NoError(t, db.Create(&course.Enrollment{CourseID: crs.ID, UserID: me.ID, UserEmail: me.Email, Role: course.RoleStudent}).Error)

	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(48 * time.Hour)
	newHandin := func(title string, pos int, deadline *time.Time) content.Content {
		cnt := content.Content{Uuid: uuid.New(), ModuleID: mod.ID, Title: title, Position: pos, Type: content.TypeAssignment}
		assert.NoError(t, db.Create(&cnt).Error)
		assert.NoError(t, db.Create(&content.Assignment{ContentID: cnt.ID, Type: content.AssignmentHandIn, MaxPoints: 5, Deadline: deadline}).Error)
		return cnt
	}
	first := newHandin("First", 1, &past)
	sub := submission.IndividualSubmission{
		Submission: submission.Submission{Uuid: uuid.New(), ContentID: first.ID, Attempt: 1, SubmittedAt: time.Now()},
		UserID:     &me.ID,
		UserEmail:  me.Email,
	}
	assert.NoError(t, db.Create(&sub).Error)
	assert.NoError(t, db.Create(&Grade{IndividualSubmissionID: &sub.ID, Score: 4, IsVisibleToStudent: true}).Error)
	second := newHandin("Second", 2, &future)

	t.Run("in progress while a later deadline is open", func(t *testing.T) {
		overview, err := svc.GetMyCourseGradeOverview(me.Uuid, crs.Uuid)
		assert.NoError(t, err)
		assert.False(t, overview.Finished)
		assert.Equal(t, 4.0, overview.ProgressScore)
		assert.Equal(t, 5.0, overview.ProgressMaxScore)

		finals, err := svc.GetStudentFinalGrades(me.Uuid, &crs.Uuid)
		assert.NoError(t, err)
		if assert.Len(t, finals, 1) {
			assert.False(t, finals[0].Finished)
			assert.Equal(t, 4.0, finals[0].ProgressScore)
			assert.Equal(t, 5.0, finals[0].ProgressMaxScore)
		}
	})

	t.Run("finished once every deadline has passed", func(t *testing.T) {
		assert.NoError(t, db.Model(&content.Assignment{}).Where("content_id = ?", second.ID).Update("deadline", past).Error)

		overview, err := svc.GetMyCourseGradeOverview(me.Uuid, crs.Uuid)
		assert.NoError(t, err)
		assert.True(t, overview.Finished)
		assert.Equal(t, 4.0, overview.ProgressScore)
		assert.Equal(t, 10.0, overview.ProgressMaxScore)
		assert.Equal(t, overview.TotalScore, overview.ProgressScore)
		assert.Equal(t, overview.TotalMaxScore, overview.ProgressMaxScore)

		finals, err := svc.GetStudentFinalGrades(me.Uuid, &crs.Uuid)
		assert.NoError(t, err)
		if assert.Len(t, finals, 1) {
			assert.True(t, finals[0].Finished)
			assert.Equal(t, overview.FinalGrade, finals[0].FinalGrade)
		}
	})
}
