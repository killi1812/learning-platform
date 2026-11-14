package submission

import (
	"context"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

const (
	reminderWindowOneDay     = 24 * time.Hour
	reminderWindowFifteenMin = 15 * time.Minute
)

// ResolveReminderWindow determines if an assignment is within a reminder window and produces the notification copy.
func ResolveReminderWindow(timeLeft time.Duration, contentTitle, courseTitle string) (ReminderType, string, string, bool) {
	if timeLeft <= 0 {
		return "", "", "", false
	}

	if timeLeft <= reminderWindowFifteenMin {
		title := fmt.Sprintf("Assignment Due Soon: %s", contentTitle)
		body := fmt.Sprintf("%q in %q is due in less than 15 minutes.", contentTitle, courseTitle)
		return ReminderFifteenMin, title, body, true
	}

	if timeLeft <= reminderWindowOneDay {
		title := fmt.Sprintf("Assignment Due Tomorrow: %s", contentTitle)
		body := fmt.Sprintf("%q in %q is due in less than 24 hours.", contentTitle, courseTitle)
		return ReminderOneDay, title, body, true
	}

	return "", "", "", false
}

type reminderCandidate struct {
	UserID uint
	Uuid   uuid.UUID
}

// processAssignmentReminders queries eligible enrolled students in a single set-based query and records reminders idempotently.
func (s *SubmissionService) processAssignmentReminders(contentID, courseID uint, reminderType ReminderType, now time.Time) ([]uuid.UUID, error) {
	var candidates []reminderCandidate
	err := s.db.Table("enrollments").
		Select("users.id as user_id, users.uuid as uuid").
		Joins("JOIN users ON users.id = enrollments.user_id").
		Where("enrollments.course_id = ? AND enrollments.role = ?", courseID, course.RoleStudent).
		Where("enrollments.deleted_at IS NULL AND users.deleted_at IS NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM individual_submissions i
			WHERE i.content_id = ? AND i.user_id = enrollments.user_id
		)`, contentID).
		Where(`NOT EXISTS (
			SELECT 1 FROM group_submissions gs
			WHERE gs.content_id = ? AND (
				gs.user_id = enrollments.user_id OR
				EXISTS (
					SELECT 1 FROM group_submission_members gsm
					WHERE gsm.group_submission_id = gs.id AND gsm.user_id = enrollments.user_id
				)
			)
		)`, contentID).
		Where(`NOT EXISTS (
			SELECT 1 FROM deadline_reminders dr
			WHERE dr.content_id = ? AND dr.user_id = enrollments.user_id AND dr.reminder_type = ?
		)`, contentID, reminderType).
		Find(&candidates).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query reminder candidates: %w", err)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	var recipientUuids []uuid.UUID
	for _, c := range candidates {
		record := DeadlineReminder{
			ContentID:    contentID,
			UserID:       c.UserID,
			ReminderType: reminderType,
			SentAt:       now,
		}
		res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		recipientUuids = append(recipientUuids, c.Uuid)
	}

	return recipientUuids, nil
}

// CheckDeadlineReminders inspects active assignments due within 24 hours or 15 minutes,
// records reminders idempotently, and notifies enrolled students who haven't submitted yet.
func (s *SubmissionService) CheckDeadlineReminders(now time.Time) (int, error) {
	s.logger.Info("Checking deadlines")
	windowMax := now.Add(reminderWindowOneDay)

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
		courseTitle := "Course"
		courseUuidStr := ""
		if asgn.Content.Module.Course != nil {
			courseTitle = asgn.Content.Module.Course.Title
			courseUuidStr = asgn.Content.Module.Course.Uuid.String()
		}

		reminderType, notifTitle, notifBody, ok := ResolveReminderWindow(timeLeft, asgn.Content.Title, courseTitle)
		if !ok {
			continue
		}

		courseID := asgn.Content.Module.CourseID
		contentID := asgn.ContentID

		recipients, err := s.processAssignmentReminders(contentID, courseID, reminderType, now)
		if err != nil {
			s.logger.Errorf("Failed to process reminders for content %d: %v", contentID, err)
			continue
		}

		if len(recipients) > 0 {
			if s.notifications != nil {
				_, err := s.notifications.NotifySystemMany(recipients, notification.NotificationContent{
					Title: notifTitle,
					Body:  notifBody,
					URL:   fmt.Sprintf("/courses/%s", courseUuidStr),
				})
				if err != nil {
					s.logger.Errorf("Failed to send deadline reminders for content %d: %v", contentID, err)
					continue
				}
			}
			totalReminders += len(recipients)
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
