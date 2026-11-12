package submission_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSubmissionService_ResolveTarget(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Group{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.Assignment{},
		&content.Quiz{},
		&content.CodingExercise{},
		&content.StudentInteraction{},
		&submission.Submission{},
		&submission.IndividualSubmission{},
		&submission.DeadlineReminder{},
		&notification.Notification{},
		&submission.UploadTicket{},
	)

	notifier := notification.NewNotifier(db, notification.NewHub(), zap.NewNop().Sugar())
	svc := submission.NewSubmissionService(db, notifier, zap.NewNop().Sugar())

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	otherCourse := &course.Course{Title: "DAT251", Term: course.TermSpring, Year: 2027}
	assert.NoError(t, db.Create(otherCourse).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	teacher := &user.User{
		Uuid:         uuid.New(),
		Username:     "teacher_1",
		Email:        "teacher1@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_1",
		Email:        "student1@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	notEnrolledStudent := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_2",
		Email:        "student2@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(teacher).Error)
	assert.NoError(t, db.Create(student).Error)
	assert.NoError(t, db.Create(notEnrolledStudent).Error)

	// Enroll student and teacher in course c
	assert.NoError(t, db.Create(&course.Enrollment{
		UserID:     student.ID,
		CourseID:   c.ID,
		Role:       course.RoleStudent,
		EnrolledAt: time.Now(),
	}).Error)
	assert.NoError(t, db.Create(&course.Enrollment{
		UserID:     teacher.ID,
		CourseID:   c.ID,
		Role:       course.RoleTeacher,
		EnrolledAt: time.Now(),
	}).Error)

	openDeadline := time.Now().Add(24 * time.Hour)
	pastDeadline := time.Now().Add(-24 * time.Hour)

	// Valid Open HandIn Assignment
	cntOpen := &content.Content{
		ModuleID: m.ID,
		Title:    "Assignment 1",
		Position: 1,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cntOpen).Error)
	asgnOpen := &content.Assignment{
		ContentID:              cntOpen.ID,
		Type:                   content.AssignmentHandIn,
		Deadline:               &openDeadline,
		AcceptsLateSubmissions: false,
	}
	assert.NoError(t, db.Create(asgnOpen).Error)

	t.Run("Happy path resolves target correctly", func(t *testing.T) {
		target, err := svc.ResolveTarget(c.Uuid, cntOpen.Uuid, student.Uuid, time.Now())
		assert.NoError(t, err)
		assert.NotNil(t, target)
		assert.Equal(t, cntOpen.ID, target.ContentID)
		assert.Equal(t, student.ID, target.UserID)
		assert.Equal(t, student.Email, target.UserEmail)
	})

	t.Run("Fails when content does not exist", func(t *testing.T) {
		_, err := svc.ResolveTarget(c.Uuid, uuid.New(), student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrContentNotFound)
	})

	t.Run("Fails when content belongs to another course", func(t *testing.T) {
		_, err := svc.ResolveTarget(otherCourse.Uuid, cntOpen.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrContentNotFound)
	})

	t.Run("Fails when content is not visible yet", func(t *testing.T) {
		future := time.Now().Add(2 * time.Hour)
		cntFuture := &content.Content{
			ModuleID:    m.ID,
			Title:       "Future Content",
			Position:    2,
			Type:        content.TypeAssignment,
			VisibleFrom: &future,
		}
		assert.NoError(t, db.Create(cntFuture).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID: cntFuture.ID,
			Type:      content.AssignmentHandIn,
		}).Error)

		_, err := svc.ResolveTarget(c.Uuid, cntFuture.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrContentNotFound)
	})

	t.Run("Fails when content visibility window has expired", func(t *testing.T) {
		past := time.Now().Add(-2 * time.Hour)
		cntExpired := &content.Content{
			ModuleID:     m.ID,
			Title:        "Expired Content",
			Position:     7,
			Type:         content.TypeAssignment,
			VisibleUntil: &past,
		}
		assert.NoError(t, db.Create(cntExpired).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID: cntExpired.ID,
			Type:      content.AssignmentHandIn,
		}).Error)

		_, err := svc.ResolveTarget(c.Uuid, cntExpired.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrContentNotFound)
	})

	t.Run("Fails when user does not exist", func(t *testing.T) {
		_, err := svc.ResolveTarget(c.Uuid, cntOpen.Uuid, uuid.New(), time.Now())
		assert.ErrorIs(t, err, submission.ErrNotEnrolled)
	})

	t.Run("Fails when content is a file", func(t *testing.T) {
		cntStatic := &content.Content{
			ModuleID: m.ID,
			Title:    "File Material",
			Position: 3,
			Type:     content.TypeFile,
		}
		assert.NoError(t, db.Create(cntStatic).Error)
		assert.NoError(t, db.Create(&content.File{
			ContentID: cntStatic.ID,
			SourceURL: "https://example.com",
		}).Error)

		_, err := svc.ResolveTarget(c.Uuid, cntStatic.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrNotAssignment)
	})

	t.Run("Fails when assignment is a Quiz", func(t *testing.T) {
		cntQuiz := &content.Content{
			ModuleID: m.ID,
			Title:    "Quiz 1",
			Position: 4,
			Type:     content.TypeAssignment,
		}
		assert.NoError(t, db.Create(cntQuiz).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID: cntQuiz.ID,
			Type:      content.AssignmentQuiz,
		}).Error)

		_, err := svc.ResolveTarget(c.Uuid, cntQuiz.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrAssignmentTypeUnsupported)
	})

	t.Run("Fails when user is not enrolled in course", func(t *testing.T) {
		_, err := svc.ResolveTarget(c.Uuid, cntOpen.Uuid, notEnrolledStudent.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrNotEnrolled)
	})

	t.Run("Fails when user is enrolled as Teacher instead of Student", func(t *testing.T) {
		_, err := svc.ResolveTarget(c.Uuid, cntOpen.Uuid, teacher.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrNotEnrolled)
	})

	t.Run("Fails when deadline passed and late submissions rejected", func(t *testing.T) {
		cntPastNoLate := &content.Content{
			ModuleID: m.ID,
			Title:    "Past Deadline No Late",
			Position: 5,
			Type:     content.TypeAssignment,
		}
		assert.NoError(t, db.Create(cntPastNoLate).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID:              cntPastNoLate.ID,
			Type:                   content.AssignmentHandIn,
			Deadline:               &pastDeadline,
			AcceptsLateSubmissions: false,
		}).Error)

		_, err := svc.ResolveTarget(c.Uuid, cntPastNoLate.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrDeadlinePassed)
	})

	t.Run("Succeeds when deadline passed but late submissions accepted", func(t *testing.T) {
		cntPastAcceptsLate := &content.Content{
			ModuleID: m.ID,
			Title:    "Past Deadline Late Accepted",
			Position: 6,
			Type:     content.TypeAssignment,
		}
		assert.NoError(t, db.Create(cntPastAcceptsLate).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID:              cntPastAcceptsLate.ID,
			Type:                   content.AssignmentHandIn,
			Deadline:               &pastDeadline,
			AcceptsLateSubmissions: true,
		}).Error)

		target, err := svc.ResolveTarget(c.Uuid, cntPastAcceptsLate.Uuid, student.Uuid, time.Now())
		assert.NoError(t, err)
		assert.NotNil(t, target)
	})

	t.Run("Fails when prerequisite module is locked, succeeds when completed", func(t *testing.T) {
		mDep := &course.Module{
			CourseID:    c.ID,
			Title:       "Week 2 (Locked)",
			Position:    2,
			DependsOnID: &m.ID,
		}
		assert.NoError(t, db.Create(mDep).Error)

		cntInLockedMod := &content.Content{
			ModuleID: mDep.ID,
			Title:    "Assignment in Locked Mod",
			Position: 1,
			Type:     content.TypeAssignment,
		}
		assert.NoError(t, db.Create(cntInLockedMod).Error)
		assert.NoError(t, db.Create(&content.Assignment{
			ContentID:              cntInLockedMod.ID,
			Type:                   content.AssignmentHandIn,
			Deadline:               &openDeadline,
			AcceptsLateSubmissions: false,
		}).Error)

		// Student has not completed all items in module m -> ErrModuleLocked
		_, err := svc.ResolveTarget(c.Uuid, cntInLockedMod.Uuid, student.Uuid, time.Now())
		assert.ErrorIs(t, err, submission.ErrModuleLocked)

		// Student completes all content items in prerequisite module m
		var allContents []content.Content
		assert.NoError(t, db.Where("module_id = ?", m.ID).Find(&allContents).Error)
		now := time.Now()
		for _, item := range allContents {
			assert.NoError(t, db.Create(&content.StudentInteraction{
				UserID:      student.ID,
				ContentID:   item.ID,
				CompletedAt: &now,
			}).Error)
		}

		// Now student can resolve target in dependent module
		target, err := svc.ResolveTarget(c.Uuid, cntInLockedMod.Uuid, student.Uuid, time.Now())
		assert.NoError(t, err)
		assert.NotNil(t, target)
	})
}

func TestSubmissionService_SaveFromMessage(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Module{},
		&content.Content{},
		&content.Assignment{},
		&content.StudentInteraction{},
		&submission.Submission{},
		&submission.IndividualSubmission{},
		&submission.DeadlineReminder{},
		&notification.Notification{},
		&submission.UploadTicket{},
	)

	notifier := notification.NewNotifier(db, notification.NewHub(), zap.NewNop().Sugar())
	svc := submission.NewSubmissionService(db, notifier, zap.NewNop().Sugar())

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	cnt := &content.Content{
		ModuleID: m.ID,
		Title:    "Assignment 1",
		Position: 1,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cnt).Error)

	deadline := time.Now().Add(2 * time.Hour)
	asgn := &content.Assignment{
		ContentID: cnt.ID,
		Type:      content.AssignmentHandIn,
		Deadline:  &deadline,
	}
	assert.NoError(t, db.Create(asgn).Error)

	u := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_save",
		Email:        "save@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(u).Error)

	t.Run("First attempt saves with attempt=1 and isLate=false", func(t *testing.T) {
		msg := submission.SubmissionMessage{
			Uuid:        uuid.New(),
			ContentID:   cnt.ID,
			UserID:      u.ID,
			UserEmail:   u.Email,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/task1.git",
			SubmittedAt: time.Now(),
		}

		saved, err := svc.SaveFromMessage(msg)
		assert.NoError(t, err)
		assert.NotNil(t, saved)
		assert.Equal(t, msg.Uuid, saved.Uuid)
		assert.Equal(t, 1, saved.Attempt)
		assert.False(t, saved.IsLate)
		assert.Equal(t, msg.PayloadURL, saved.PayloadURL)
		assert.Equal(t, u.Email, saved.UserEmail)

		var inter content.StudentInteraction
		assert.NoError(t, db.Where("user_id = ? AND content_id = ?", u.ID, cnt.ID).First(&inter).Error)
		assert.NotNil(t, inter.CompletedAt)
	})

	t.Run("Second attempt saves with attempt=2", func(t *testing.T) {
		// Ensure first attempt exists so subtest is completely isolated
		var existingCount int64
		assert.NoError(t, db.Model(&submission.IndividualSubmission{}).Where("content_id = ? AND user_id = ?", cnt.ID, u.ID).Count(&existingCount).Error)
		if existingCount == 0 {
			_, err := svc.SaveFromMessage(submission.SubmissionMessage{
				Uuid:        uuid.New(),
				ContentID:   cnt.ID,
				UserID:      u.ID,
				UserEmail:   u.Email,
				PayloadType: submission.PayloadTypeURL,
				PayloadURL:  "https://git.app.uib.no/student/task1.git",
				SubmittedAt: time.Now(),
			})
			assert.NoError(t, err)
		}

		msg := submission.SubmissionMessage{
			Uuid:        uuid.New(),
			ContentID:   cnt.ID,
			UserID:      u.ID,
			UserEmail:   u.Email,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/task1-v2.git",
			SubmittedAt: time.Now(),
		}

		saved, err := svc.SaveFromMessage(msg)
		assert.NoError(t, err)
		assert.NotNil(t, saved)
		assert.Equal(t, 2, saved.Attempt)
		assert.False(t, saved.IsLate)
	})

	t.Run("Late submission has isLate=true", func(t *testing.T) {
		msg := submission.SubmissionMessage{
			Uuid:        uuid.New(),
			ContentID:   cnt.ID,
			UserID:      u.ID,
			UserEmail:   u.Email,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/task1-late.git",
			SubmittedAt: time.Now().Add(4 * time.Hour), // After deadline
		}

		saved, err := svc.SaveFromMessage(msg)
		assert.NoError(t, err)
		assert.NotNil(t, saved)
		assert.True(t, saved.IsLate)
	})

	t.Run("Duplicate message delivery is idempotent", func(t *testing.T) {
		duplicateUUID := uuid.New()
		msg := submission.SubmissionMessage{
			Uuid:        duplicateUUID,
			ContentID:   cnt.ID,
			UserID:      u.ID,
			UserEmail:   u.Email,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/task1-dup.git",
			SubmittedAt: time.Now(),
		}

		first, err := svc.SaveFromMessage(msg)
		assert.NoError(t, err)

		var notifsAfterFirst []notification.Notification
		assert.NoError(t, db.Where("user_id = ? AND title = ?", u.ID, "Assignment Saved").Find(&notifsAfterFirst).Error)
		firstCount := len(notifsAfterFirst)
		assert.Greater(t, firstCount, 0)
		assert.Equal(t, "/courses/"+c.Uuid.String(), notifsAfterFirst[len(notifsAfterFirst)-1].URL)

		second, err := svc.SaveFromMessage(msg)
		assert.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, first.Attempt, second.Attempt)

		var notifsAfterSecond []notification.Notification
		assert.NoError(t, db.Where("user_id = ? AND title = ?", u.ID, "Assignment Saved").Find(&notifsAfterSecond).Error)
		assert.Equal(t, firstCount, len(notifsAfterSecond), "idempotent retry must not duplicate notification")
	})

	t.Run("NotifySubmitted sends queued notification", func(t *testing.T) {
		target := &submission.SubmissionTarget{
			ContentID:    cnt.ID,
			ContentTitle: "Assignment 1",
			CourseUuid:   c.Uuid,
			UserID:       u.ID,
			UserUuid:     u.Uuid,
		}
		svc.NotifySubmitted(target)

		var notifs []notification.Notification
		assert.NoError(t, db.Where("user_id = ? AND title = ?", u.ID, "Assignment Submitted").Find(&notifs).Error)
		assert.NotEmpty(t, notifs)
		assert.Contains(t, notifs[len(notifs)-1].Body, "queued for processing")
	})

	t.Run("NotifySubmissionFailed sends failure notification", func(t *testing.T) {
		svc.NotifySubmissionFailed(u.Uuid, c.Uuid, "Assignment 1")

		var notifs []notification.Notification
		assert.NoError(t, db.Where("user_id = ? AND title = ?", u.ID, "Assignment Submission Failed").Find(&notifs).Error)
		assert.NotEmpty(t, notifs)
		assert.Contains(t, notifs[len(notifs)-1].Body, "could not process your submission")
		assert.Contains(t, notifs[len(notifs)-1].Body, "contact your professor")
	})

	t.Run("NotifySubmissionFailed with nil userUuid logs error and skips", func(t *testing.T) {
		assert.NotPanics(t, func() {
			svc.NotifySubmissionFailed(uuid.Nil, c.Uuid, "Assignment 1")
		})
	})

	t.Run("NotifySubmissionFailed with nil courseUuid falls back to dashboard URL", func(t *testing.T) {
		svc.NotifySubmissionFailed(u.Uuid, uuid.Nil, "Assignment 1")

		var notifs []notification.Notification
		assert.NoError(t, db.Where("user_id = ? AND title = ?", u.ID, "Assignment Submission Failed").Find(&notifs).Error)
		assert.NotEmpty(t, notifs)
		assert.Equal(t, "/dashboard", notifs[len(notifs)-1].URL)
	})

	t.Run("NotifySubmitted with nil target or nil userUuid logs error and skips", func(t *testing.T) {
		assert.NotPanics(t, func() {
			svc.NotifySubmitted(nil)
			svc.NotifySubmitted(&submission.SubmissionTarget{UserUuid: uuid.Nil})
		})
	})

	t.Run("Fails when assignment for content does not exist", func(t *testing.T) {
		msg := submission.SubmissionMessage{
			Uuid:        uuid.New(),
			ContentID:   999999,
			UserID:      u.ID,
			UserEmail:   u.Email,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/missing.git",
			SubmittedAt: time.Now(),
		}

		saved, err := svc.SaveFromMessage(msg)
		assert.Error(t, err)
		assert.Nil(t, saved)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSubmissionService_FileSubmissionWorkflow(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Group{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.Assignment{},
		&content.StudentInteraction{},
		&submission.Submission{},
		&submission.IndividualSubmission{},
		&submission.DeadlineReminder{},
		&notification.Notification{},
		&submission.UploadTicket{},
	)

	notifier := notification.NewNotifier(db, notification.NewHub(), zap.NewNop().Sugar())
	svc := submission.NewSubmissionService(db, notifier, zap.NewNop().Sugar())

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	deadline := time.Now().Add(24 * time.Hour)
	cnt := &content.Content{
		ModuleID: m.ID,
		Title:    "Lab 1",
		Position: 1,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cnt).Error)
	asgn := &content.Assignment{
		ContentID:              cnt.ID,
		Type:                   content.AssignmentHandIn,
		Deadline:               &deadline,
		AcceptsLateSubmissions: false,
	}
	assert.NoError(t, db.Create(asgn).Error)

	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_file",
		Email:        "student_file@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	otherStudent := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_other",
		Email:        "student_other@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(student).Error)
	assert.NoError(t, db.Create(otherStudent).Error)

	assert.NoError(t, db.Create(&course.Enrollment{
		UserID:     student.ID,
		CourseID:   c.ID,
		Role:       course.RoleStudent,
		EnrolledAt: time.Now(),
	}).Error)

	now := time.Now()

	// 1. InitFileSubmission
	ticket, err := svc.InitFileSubmission(c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.NotNil(t, ticket)
	assert.Equal(t, student.ID, ticket.UserID)
	assert.False(t, ticket.Consumed)
	assert.True(t, ticket.ExpiresAt.After(now))

	// 2. ValidateAndExtendTicket success
	validated, err := svc.ValidateAndExtendTicket(ticket.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.Equal(t, ticket.Uuid, validated.Uuid)

	// Validate with non-existent ticket
	_, err = svc.ValidateAndExtendTicket(uuid.New(), c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.ErrorIs(t, err, submission.ErrTicketNotFound)

	// Validate expired ticket
	expiredTicket := &submission.UploadTicket{
		Uuid:        uuid.New(),
		CourseID:    c.ID,
		ContentID:   cnt.ID,
		UserID:      student.ID,
		UserEmail:   student.Email,
		SubmittedAt: now.Add(-2 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour),
		Consumed:    false,
	}
	assert.NoError(t, db.Create(expiredTicket).Error)
	_, err = svc.ValidateAndExtendTicket(expiredTicket.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.ErrorIs(t, err, submission.ErrTicketExpired)

	// Validate ticket expiring soon extends expiration
	expiringSoonTicket := &submission.UploadTicket{
		Uuid:        uuid.New(),
		CourseID:    c.ID,
		ContentID:   cnt.ID,
		UserID:      student.ID,
		UserEmail:   student.Email,
		SubmittedAt: now,
		ExpiresAt:   now.Add(5 * time.Minute),
		Consumed:    false,
	}
	assert.NoError(t, db.Create(expiringSoonTicket).Error)
	extendedTicket, err := svc.ValidateAndExtendTicket(expiringSoonTicket.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.True(t, extendedTicket.ExpiresAt.After(now.Add(10*time.Minute)))

	// 3. ConsumeUploadTicket
	assert.NoError(t, svc.ConsumeUploadTicket(ticket.Uuid))
	assert.ErrorIs(t, svc.ConsumeUploadTicket(ticket.Uuid), submission.ErrTicketConsumed)
	assert.ErrorIs(t, svc.ConsumeUploadTicket(uuid.New()), submission.ErrTicketNotFound)
	_, err = svc.ValidateAndExtendTicket(ticket.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.ErrorIs(t, err, submission.ErrTicketConsumed)

	// InitFileSubmission supersedes any previously unconsumed ticket for same student & assignment
	ticket1, err := svc.InitFileSubmission(c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.False(t, ticket1.Consumed)

	ticket2, err := svc.InitFileSubmission(c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.False(t, ticket2.Consumed)

	// ticket1 was superseded (marked consumed)
	_, err = svc.ValidateAndExtendTicket(ticket1.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.ErrorIs(t, err, submission.ErrTicketConsumed)

	// ticket2 is valid
	val2, err := svc.ValidateAndExtendTicket(ticket2.Uuid, c.Uuid, cnt.Uuid, student.Uuid, now)
	assert.NoError(t, err)
	assert.Equal(t, ticket2.Uuid, val2.Uuid)

	// 4. SaveFromMessage with PayloadTypeFile
	fileMsg := submission.SubmissionMessage{
		Uuid:        ticket.Uuid,
		ContentID:   cnt.ID,
		UserID:      student.ID,
		UserEmail:   student.Email,
		PayloadType: submission.PayloadTypeFile,
		FileName:    "lab1.pdf",
		FileSize:    2048,
		ContentType: "application/pdf",
		StorageKey:  "submissions/test/lab1.pdf",
		SubmittedAt: now,
	}
	savedSub, err := svc.SaveFromMessage(fileMsg)
	assert.NoError(t, err)
	assert.NotNil(t, savedSub)
	assert.Equal(t, "lab1.pdf", savedSub.FileName)
	assert.Equal(t, int64(2048), savedSub.FileSize)
	assert.Equal(t, "application/pdf", savedSub.ContentType)
	assert.Equal(t, "submissions/test/lab1.pdf", savedSub.StorageKey)
	assert.Equal(t, 1, savedSub.Attempt)
	assert.False(t, savedSub.IsLate)

	// 5. ListMySubmissions
	mySubs, err := svc.ListMySubmissions(c.Uuid, cnt.Uuid, student.Uuid)
	assert.NoError(t, err)
	assert.Len(t, mySubs, 1)
	assert.Equal(t, "lab1.pdf", mySubs[0].FileName)

	// ListMySubmissions for not enrolled user
	_, err = svc.ListMySubmissions(c.Uuid, cnt.Uuid, uuid.New())
	assert.Error(t, err)

	// ListMySubmissions for missing content
	_, err = svc.ListMySubmissions(c.Uuid, uuid.New(), student.Uuid)
	assert.ErrorIs(t, err, submission.ErrContentNotFound)

	// 6. ListCourseAssignmentSubmissions
	allSubs, err := svc.ListCourseAssignmentSubmissions(c.Uuid, cnt.Uuid)
	assert.NoError(t, err)
	assert.Len(t, allSubs, 1)

	// 7. GetSubmissionForDownload
	// Owner can download
	dl, err := svc.GetSubmissionForDownload(c.Uuid, cnt.Uuid, savedSub.Uuid, student.Uuid, false)
	assert.NoError(t, err)
	assert.Equal(t, savedSub.Uuid, dl.Uuid)

	// Privileged user (teacher) can download
	dlTeacher, err := svc.GetSubmissionForDownload(c.Uuid, cnt.Uuid, savedSub.Uuid, uuid.New(), true)
	assert.NoError(t, err)
	assert.Equal(t, savedSub.Uuid, dlTeacher.Uuid)

	// Another student cannot download
	_, err = svc.GetSubmissionForDownload(c.Uuid, cnt.Uuid, savedSub.Uuid, otherStudent.Uuid, false)
	assert.ErrorIs(t, err, submission.ErrUnauthorizedSubmission)

	// Non-existent user cannot download
	_, err = svc.GetSubmissionForDownload(c.Uuid, cnt.Uuid, savedSub.Uuid, uuid.New(), false)
	assert.ErrorIs(t, err, submission.ErrUnauthorizedSubmission)

	// Unknown submission
	_, err = svc.GetSubmissionForDownload(c.Uuid, cnt.Uuid, uuid.New(), student.Uuid, false)
	assert.ErrorIs(t, err, submission.ErrSubmissionNotFound)

	// Content not found
	_, err = svc.GetSubmissionForDownload(uuid.New(), cnt.Uuid, savedSub.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, submission.ErrContentNotFound)

	// ListCourseAssignmentSubmissions with unknown course
	_, err = svc.ListCourseAssignmentSubmissions(uuid.New(), cnt.Uuid)
	assert.ErrorIs(t, err, submission.ErrContentNotFound)

	// InitFileSubmission with unknown content
	_, err = svc.InitFileSubmission(uuid.New(), cnt.Uuid, student.Uuid, now)
	assert.ErrorIs(t, err, submission.ErrContentNotFound)

	// 8. Model BeforeCreate checks
	ticketNilUuid := &submission.UploadTicket{
		CourseID:    c.ID,
		ContentID:   cnt.ID,
		UserID:      student.ID,
		UserEmail:   student.Email,
		SubmittedAt: now,
		ExpiresAt:   now.Add(30 * time.Minute),
	}
	assert.NoError(t, db.Create(ticketNilUuid).Error)
	assert.NotEqual(t, uuid.Nil, ticketNilUuid.Uuid)

	customUuid := uuid.New()
	subCustom := &submission.IndividualSubmission{
		Submission: submission.Submission{
			Uuid:        customUuid,
			ContentID:   cnt.ID,
			Attempt:     2,
			SubmittedAt: now,
		},
		UserID:    &student.ID,
		UserEmail: student.Email,
	}
	assert.NoError(t, db.Create(subCustom).Error)
	assert.Equal(t, customUuid, subCustom.Uuid)
}
