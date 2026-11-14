package submission_test

import (
	"context"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSubmissionService_CheckDeadlineReminders(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Group{},
		&course.Module{},
		&content.Content{},
		&content.Assignment{},
		&submission.Submission{},
		&submission.IndividualSubmission{},
		&submission.GroupSubmission{},
		&submission.DeadlineReminder{},
		&notification.Notification{},
	)

	logger := zap.NewNop().Sugar()
	hub := notification.NewHub()
	notifier := notification.NewNotifier(db, hub, logger)
	svc := submission.NewSubmissionService(db, notifier, logger)

	c := &course.Course{Title: "Operating Systems", Term: course.TermAutumn, Year: 2026}
	require.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	require.NoError(t, db.Create(m).Error)

	student1 := &user.User{Uuid: uuid.New(), Username: "s1", Email: "s1@example.com"}
	student2 := &user.User{Uuid: uuid.New(), Username: "s2", Email: "s2@example.com"}
	teacher := &user.User{Uuid: uuid.New(), Username: "prof", Email: "prof@example.com"}
	require.NoError(t, db.Create(student1).Error)
	require.NoError(t, db.Create(student2).Error)
	require.NoError(t, db.Create(teacher).Error)

	require.NoError(t, db.Create(&course.Enrollment{UserID: student1.ID, CourseID: c.ID, Role: course.RoleStudent}).Error)
	require.NoError(t, db.Create(&course.Enrollment{UserID: student2.ID, CourseID: c.ID, Role: course.RoleStudent}).Error)
	require.NoError(t, db.Create(&course.Enrollment{UserID: teacher.ID, CourseID: c.ID, Role: course.RoleTeacher}).Error)

	now := time.Now().UTC()

	// 1. Assignment due in 12 hours (24H window)
	due12h := now.Add(12 * time.Hour)
	cnt24h := &content.Content{ModuleID: m.ID, Title: "OS Lab 1", Position: 1, Type: content.TypeAssignment}
	require.NoError(t, db.Create(cnt24h).Error)
	asgn24h := &content.Assignment{ContentID: cnt24h.ID, Deadline: &due12h, Type: content.AssignmentHandIn}
	require.NoError(t, db.Create(asgn24h).Error)

	// Student 2 has already submitted OS Lab 1
	subS2 := submission.IndividualSubmission{
		Submission: submission.Submission{
			Uuid:        uuid.New(),
			ContentID:   cnt24h.ID,
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.example.com/s2/lab1",
			SubmittedAt: now,
		},
		UserID:    &student2.ID,
		UserEmail: student2.Email,
	}
	require.NoError(t, db.Create(&subS2).Error)

	t.Run("24H window notifies unsubmitted student and skips submitted student", func(t *testing.T) {
		remindersSent, err := svc.CheckDeadlineReminders(now)
		require.NoError(t, err)
		assert.Equal(t, 1, remindersSent) // only student1

		var s1Notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", student1.ID).Find(&s1Notifs).Error)
		require.Len(t, s1Notifs, 1)
		assert.Equal(t, "Assignment Due Tomorrow: OS Lab 1", s1Notifs[0].Title)
		assert.Contains(t, s1Notifs[0].Body, "less than 24 hours")

		var s2Notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", student2.ID).Find(&s2Notifs).Error)
		assert.Empty(t, s2Notifs)

		// Second run is idempotent and sends 0 reminders
		remindersAgain, err := svc.CheckDeadlineReminders(now)
		require.NoError(t, err)
		assert.Zero(t, remindersAgain)
	})

	t.Run("15M window notifies student when deadline is in 10 minutes", func(t *testing.T) {
		// Advance time so that OS Lab 1 is now 10 minutes from deadline
		tenMinutesBefore := due12h.Add(-10 * time.Minute)

		remindersSent, err := svc.CheckDeadlineReminders(tenMinutesBefore)
		require.NoError(t, err)
		assert.Equal(t, 1, remindersSent)

		var s1Notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", student1.ID).Order("id asc").Find(&s1Notifs).Error)
		require.Len(t, s1Notifs, 2)
		assert.Equal(t, "Assignment Due Soon: OS Lab 1", s1Notifs[1].Title)
		assert.Contains(t, s1Notifs[1].Body, "less than 15 minutes")

		// Re-run at same time sends nothing
		remindersAgain, err := svc.CheckDeadlineReminders(tenMinutesBefore)
		require.NoError(t, err)
		assert.Zero(t, remindersAgain)
	})

	t.Run("Past deadline and far future deadline are skipped", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		cntPast := &content.Content{ModuleID: m.ID, Title: "Past Lab", Position: 2, Type: content.TypeAssignment}
		require.NoError(t, db.Create(cntPast).Error)
		asgnPast := &content.Assignment{ContentID: cntPast.ID, Deadline: &pastTime, Type: content.AssignmentHandIn}
		require.NoError(t, db.Create(asgnPast).Error)

		farTime := now.Add(48 * time.Hour)
		cntFar := &content.Content{ModuleID: m.ID, Title: "Far Lab", Position: 3, Type: content.TypeAssignment}
		require.NoError(t, db.Create(cntFar).Error)
		asgnFar := &content.Assignment{ContentID: cntFar.ID, Deadline: &farTime, Type: content.AssignmentHandIn}
		require.NoError(t, db.Create(asgnFar).Error)

		remindersSent, err := svc.CheckDeadlineReminders(now)
		require.NoError(t, err)
		assert.Zero(t, remindersSent)
	})

	t.Run("Group submission skips deadline reminder", func(t *testing.T) {
		dueIn5h := now.Add(5 * time.Hour)
		cntGrp := &content.Content{ModuleID: m.ID, Title: "Group Lab", Position: 4, Type: content.TypeAssignment}
		require.NoError(t, db.Create(cntGrp).Error)
		asgnGrp := &content.Assignment{ContentID: cntGrp.ID, Deadline: &dueIn5h, Type: content.AssignmentHandIn}
		require.NoError(t, db.Create(asgnGrp).Error)

		student3 := &user.User{Uuid: uuid.New(), Username: "s3", Email: "s3@example.com"}
		require.NoError(t, db.Create(student3).Error)
		require.NoError(t, db.Create(&course.Enrollment{UserID: student3.ID, CourseID: c.ID, Role: course.RoleStudent}).Error)

		grp := &course.Group{CourseID: c.ID, Name: "OS Group", Members: []user.User{*student3}}
		require.NoError(t, db.Create(grp).Error)

		grpSub := submission.GroupSubmission{
			Submission: submission.Submission{
				Uuid:        uuid.New(),
				ContentID:   cntGrp.ID,
				PayloadType: submission.PayloadTypeURL,
				PayloadURL:  "https://git.example.com/grp/lab",
				SubmittedAt: now,
			},
			GroupID: &grp.ID,
			UserID:  &student3.ID,
			Members: []user.User{*student3},
		}
		require.NoError(t, db.Create(&grpSub).Error)

		remindersSent, err := svc.CheckDeadlineReminders(now)
		require.NoError(t, err)
		// student1 will get 24h reminder for Group Lab, but student3 who is in the group that submitted will not!
		var s3Notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ? AND title LIKE ?", student3.ID, "%Group Lab%").Find(&s3Notifs).Error)
		assert.Empty(t, s3Notifs)
		assert.GreaterOrEqual(t, remindersSent, 1)
	})

	t.Run("Edge cases with nil notifications and empty targets", func(t *testing.T) {
		nilNotifSvc := submission.NewSubmissionService(db, nil, logger)
		assert.NotPanics(t, func() {
			nilNotifSvc.NotifySubmitted(nil)
			nilNotifSvc.NotifySubmitted(&submission.SubmissionTarget{})
			nilNotifSvc.NotifySubmissionFailed(uuid.Nil, uuid.Nil, "")
		})
	})
}

func TestDeadlineReminderWorker_RunAndStop(t *testing.T) {
	mockSvc := &mockSubmissionService{
		checkDeadlineRemindersFunc: func(now time.Time) (int, error) {
			return 0, assert.AnError
		},
	}
	logger := zap.NewNop().Sugar()
	worker := submission.NewDeadlineReminderWorker(mockSvc, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop on canceled context")
	}
}

func TestSubmissionService_CheckDeadlineReminders_DBError(t *testing.T) {
	closedDb := testutil.NewSqliteTestDb(t, &content.Assignment{})
	sqlDB, err := closedDb.DB()
	require.NoError(t, err)
	_ = sqlDB.Close()

	svc := submission.NewSubmissionService(closedDb, nil, zap.NewNop().Sugar())
	_, err = svc.CheckDeadlineReminders(time.Now())
	assert.Error(t, err)
}

func TestResolveReminderWindow(t *testing.T) {
	tests := []struct {
		name         string
		timeLeft     time.Duration
		expectedType submission.ReminderType
		expectedOk   bool
		containsBody string
	}{
		{
			name:       "Negative time left (past deadline)",
			timeLeft:   -5 * time.Minute,
			expectedOk: false,
		},
		{
			name:       "Zero time left (exact deadline)",
			timeLeft:   0,
			expectedOk: false,
		},
		{
			name:         "5 minutes left (15M window)",
			timeLeft:     5 * time.Minute,
			expectedType: submission.ReminderFifteenMin,
			expectedOk:   true,
			containsBody: "less than 15 minutes",
		},
		{
			name:         "Exactly 15 minutes left (15M window)",
			timeLeft:     15 * time.Minute,
			expectedType: submission.ReminderFifteenMin,
			expectedOk:   true,
			containsBody: "less than 15 minutes",
		},
		{
			name:         "16 minutes left (24H window)",
			timeLeft:     16 * time.Minute,
			expectedType: submission.ReminderOneDay,
			expectedOk:   true,
			containsBody: "less than 24 hours",
		},
		{
			name:         "12 hours left (24H window)",
			timeLeft:     12 * time.Hour,
			expectedType: submission.ReminderOneDay,
			expectedOk:   true,
			containsBody: "less than 24 hours",
		},
		{
			name:         "Exactly 24 hours left (24H window)",
			timeLeft:     24 * time.Hour,
			expectedType: submission.ReminderOneDay,
			expectedOk:   true,
			containsBody: "less than 24 hours",
		},
		{
			name:       "25 hours left (beyond reminder window)",
			timeLeft:   25 * time.Hour,
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remType, title, body, ok := submission.ResolveReminderWindow(tt.timeLeft, "Lab 1", "OS")
			assert.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				assert.Equal(t, tt.expectedType, remType)
				assert.Contains(t, title, "Lab 1")
				assert.Contains(t, body, tt.containsBody)
				assert.Contains(t, body, "OS")
			}
		})
	}
}
