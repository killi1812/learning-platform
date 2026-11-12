package submission

import (
	"context"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

// CheckDeadlineReminders inspects active assignments due within 24 hours or 15 minutes,
// records reminders idempotently, and notifies enrolled students who haven't submitted yet.
func (s *SubmissionService) CheckDeadlineReminders(now time.Time) (int, error) {
	s.logger.Info("Checking deadlines")
	windowMax := now.Add(24 * time.Hour)

	var assignments []content.Assignment
	err := s.db.
		Joins("JOIN contents ON contents.id = assignments.content_id").
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Preload("Content.Module.Course").
		Where("assignments.deadline IS NOT NULL AND assignments.deadline > ? AND assignments.deadline <= ?", now, windowMax).
		Where("(contents.visible_from IS NULL OR contents.visible_from <= ?)", now).
		Where("(contents.visible_until IS NULL OR contents.visible_until >= ?)", now).
		Find(&assignments).Error
	if err != nil {
		return 0, fmt.Errorf("failed to query upcoming assignments: %w", err)
	}

	totalReminders := 0

	for _, asgn := range assignments {
		if asgn.Content == nil || asgn.Deadline == nil || asgn.Content.Module == nil {
			continue
		}

		timeLeft := asgn.Deadline.Sub(now)
		if timeLeft <= 0 {
			continue
		}

		var reminderType ReminderType
		var notifTitle string
		var notifBody string

		courseTitle := "Course"
		courseUuidStr := ""
		if asgn.Content.Module.Course != nil {
			courseTitle = asgn.Content.Module.Course.Title
			courseUuidStr = asgn.Content.Module.Course.Uuid.String()
		}

		if timeLeft <= 15*time.Minute {
			reminderType = ReminderFifteenMin
			notifTitle = fmt.Sprintf("Assignment Due Soon: %s", asgn.Content.Title)
			notifBody = fmt.Sprintf("%q in %q is due in less than 15 minutes.", asgn.Content.Title, courseTitle)
		} else if timeLeft <= 24*time.Hour {
			reminderType = ReminderOneDay
			notifTitle = fmt.Sprintf("Assignment Due Tomorrow: %s", asgn.Content.Title)
			notifBody = fmt.Sprintf("%q in %q is due in less than 24 hours.", asgn.Content.Title, courseTitle)
		} else {
			continue
		}

		courseID := asgn.Content.Module.CourseID
		contentID := asgn.ContentID

		// Find enrolled students in this course
		var enrollments []course.Enrollment
		if err := s.db.Preload("User").
			Where("course_id = ? AND role = ?", courseID, course.RoleStudent).
			Find(&enrollments).Error; err != nil {
			s.logger.Errorf("Failed to query enrollments for course %d: %v", courseID, err)
			continue
		}

		var recipientUuids []uuid.UUID

		for _, enr := range enrollments {
			if enr.User == nil {
				var u user.User
				if err := s.db.First(&u, enr.UserID).Error; err != nil {
					continue
				}
				enr.User = &u
			}

			// Check if already submitted via IndividualSubmission
			var indCount int64
			if err := s.db.Model(&IndividualSubmission{}).
				Where("content_id = ? AND user_id = ?", contentID, enr.UserID).
				Count(&indCount).Error; err != nil {
				continue
			}
			if indCount > 0 {
				continue
			}

			// Check if already submitted via GroupSubmission (if group submission exists for user in snapshot or user submitted)
			var grpCount int64
			if err := s.db.Table("group_submissions").
				Where("content_id = ? AND (user_id = ? OR id IN (SELECT group_submission_id FROM group_submission_members WHERE user_id = ?))",
					contentID, enr.UserID, enr.UserID).
				Count(&grpCount).Error; err != nil {
				continue
			}
			if grpCount > 0 {
				continue
			}

			// Check if reminder was already recorded
			var remCount int64
			if err := s.db.Model(&DeadlineReminder{}).
				Where("content_id = ? AND user_id = ? AND reminder_type = ?", contentID, enr.UserID, reminderType).
				Count(&remCount).Error; err != nil {
				continue
			}
			if remCount > 0 {
				continue
			}

			// Insert reminder record with on conflict do nothing
			reminderRecord := DeadlineReminder{
				ContentID:    contentID,
				UserID:       enr.UserID,
				ReminderType: reminderType,
				SentAt:       now,
			}
			res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&reminderRecord)
			if res.Error != nil || res.RowsAffected == 0 {
				continue
			}

			recipientUuids = append(recipientUuids, enr.User.Uuid)
		}

		if len(recipientUuids) > 0 {
			_, err := s.notifications.NotifySystemMany(recipientUuids, notification.NotificationContent{
				Title: notifTitle,
				Body:  notifBody,
				URL:   fmt.Sprintf("/courses/%s", courseUuidStr),
			})
			if err != nil {
				s.logger.Errorf("Failed to send deadline reminders for content %d: %v", contentID, err)
			} else {
				totalReminders += len(recipientUuids)
			}
		}
	}

	return totalReminders, nil
}

// DeadlineReminderWorker periodically evaluates assignment deadlines and triggers notifications.
type DeadlineReminderWorker struct {
	service ISubmissionService
	logger  *zap.SugaredLogger
}

// NewDeadlineReminderWorker constructs a new DeadlineReminderWorker.
func NewDeadlineReminderWorker(service ISubmissionService, logger *zap.SugaredLogger) app.Worker {
	return &DeadlineReminderWorker{
		service: service,
		logger:  logger,
	}
}

// Run executes the periodic deadline reminder evaluation loop until context is canceled.
func (w *DeadlineReminderWorker) Run(ctx context.Context) {
	w.logger.Info("Starting assignment deadline reminder worker")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	if _, err := w.service.CheckDeadlineReminders(time.Now().UTC()); err != nil {
		w.logger.Errorf("Initial deadline reminder check error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping assignment deadline reminder worker")
			return
		case t := <-ticker.C:
			if _, err := w.service.CheckDeadlineReminders(t.UTC()); err != nil {
				w.logger.Errorf("Periodic deadline reminder check error: %v", err)
			}
		}
	}
}
