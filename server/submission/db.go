package submission

import (
	"errors"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrContentNotFound           = errors.New("assignment not found")
	ErrNotAssignment             = errors.New("content is not an assignment")
	ErrAssignmentTypeUnsupported = errors.New("submissions for this assignment type are not supported yet")
	ErrNotEnrolled               = errors.New("user is not enrolled as a student in this course")
	ErrDeadlinePassed            = errors.New("deadline has passed and late submissions are not accepted")
	ErrModuleLocked              = errors.New("module is locked because prerequisite module has not been completed")
)

// SubmissionTarget contains database primary IDs for publishing the submission message.
type SubmissionTarget struct {
	ContentID    uint
	ContentUuid  uuid.UUID
	ContentTitle string
	CourseUuid   uuid.UUID
	UserID       uint
	UserUuid     uuid.UUID
	UserEmail    string
}

// ISubmissionService provides domain validation and database persistence for submissions.
type ISubmissionService interface {
	ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*SubmissionTarget, error)
	SaveFromMessage(msg SubmissionMessage) (*IndividualSubmission, error)
	NotifySubmitted(target *SubmissionTarget)
	NotifySubmissionFailed(userUuid, courseUuid uuid.UUID, contentTitle string)
	CheckDeadlineReminders(now time.Time) (int, error)
}

// SubmissionService implements ISubmissionService using GORM.
type SubmissionService struct {
	db            *gorm.DB
	notifications notification.INotificationService
	logger        *zap.SugaredLogger
}

// NewSubmissionService constructs a new SubmissionService.
func NewSubmissionService(db *gorm.DB, notifications notification.INotificationService, logger *zap.SugaredLogger) ISubmissionService {
	return &SubmissionService{
		db:            db,
		notifications: notifications,
		logger:        logger,
	}
}

// ResolveTarget validates content existence, type, visibility, deadline constraints, and user enrollment on the hot path.
func (s *SubmissionService) ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*SubmissionTarget, error) {
	// 1. Fetch content joined with its module and course to verify association
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Preload("Assignment").
		Preload("Module").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	if cnt.VisibleFrom != nil && submittedAt.Before(*cnt.VisibleFrom) {
		return nil, ErrContentNotFound
	}
	if cnt.VisibleUntil != nil && submittedAt.After(*cnt.VisibleUntil) {
		return nil, ErrContentNotFound
	}

	if cnt.Type != content.TypeAssignment || cnt.Assignment == nil {
		return nil, ErrNotAssignment
	}

	if cnt.Assignment.Type != content.AssignmentHandIn {
		return nil, ErrAssignmentTypeUnsupported
	}

	if cnt.Assignment.Deadline != nil && submittedAt.After(*cnt.Assignment.Deadline) && !cnt.Assignment.AcceptsLateSubmissions {
		return nil, ErrDeadlinePassed
	}

	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotEnrolled
		}
		return nil, err
	}

	if cnt.Module != nil && cnt.Module.DependsOnID != nil {
		depID := *cnt.Module.DependsOnID
		var totalContents int64
		if err := s.db.Model(&content.Content{}).Where("module_id = ?", depID).Count(&totalContents).Error; err != nil {
			return nil, err
		}
		if totalContents > 0 {
			var completedContents int64
			err := s.db.Model(&content.StudentInteraction{}).
				Joins("JOIN contents ON contents.id = student_interactions.content_id").
				Where("contents.module_id = ? AND student_interactions.user_id = ? AND student_interactions.completed_at IS NOT NULL", depID, u.ID).
				Count(&completedContents).Error
			if err != nil {
				return nil, err
			}
			if completedContents < totalContents {
				return nil, ErrModuleLocked
			}
		}
	}

	var enrollment course.Enrollment
	err = s.db.
		Joins("JOIN courses ON courses.id = enrollments.course_id").
		Where("courses.uuid = ? AND enrollments.user_id = ? AND enrollments.role = ?", courseUuid, u.ID, course.RoleStudent).
		First(&enrollment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotEnrolled
		}
		return nil, err
	}

	return &SubmissionTarget{
		ContentID:    cnt.ID,
		ContentUuid:  cnt.Uuid,
		ContentTitle: cnt.Title,
		CourseUuid:   courseUuid,
		UserID:       u.ID,
		UserUuid:     u.Uuid,
		UserEmail:    u.Email,
	}, nil
}

// SaveFromMessage persists an enqueued submission into the database. Idempotent on msg.Uuid.
func (s *SubmissionService) SaveFromMessage(msg SubmissionMessage) (*IndividualSubmission, error) {
	// Fast check for duplicate delivery revers logic return if submission exists
	var existing IndividualSubmission
	if err := s.db.Where("uuid = ?", msg.Uuid).First(&existing).Error; err == nil {
		s.logger.Debugf("Submission %s already exists, skipping duplicate message", msg.Uuid)
		return &existing, nil
	}

	var savedSub IndividualSubmission
	var created bool
	var courseUuid uuid.UUID
	var contentTitle string
	var studentUserUuid uuid.UUID

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Re-check within transaction
		if err := tx.Where("uuid = ?", msg.Uuid).First(&savedSub).Error; err == nil {
			return nil
		}

		// Fetch assignment to compute isLate
		var asgn content.Assignment
		if err := tx.Where("content_id = ?", msg.ContentID).First(&asgn).Error; err != nil {
			return fmt.Errorf("assignment for content %d not found: %w", msg.ContentID, err)
		}

		// Compute attempt number for this student on this assignment
		var maxAttempt int
		row := tx.Model(&IndividualSubmission{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("content_id = ? AND user_id = ?", msg.ContentID, msg.UserID).
			Select("COALESCE(MAX(attempt), 0)").
			Row()
		if err := row.Scan(&maxAttempt); err != nil {
			return fmt.Errorf("failed to query max attempt: %w", err)
		}

		isLate := asgn.Deadline != nil && msg.SubmittedAt.After(*asgn.Deadline)

		savedSub = IndividualSubmission{
			Submission: Submission{
				Uuid:        msg.Uuid,
				ContentID:   msg.ContentID,
				Attempt:     maxAttempt + 1,
				IsLate:      isLate,
				PayloadType: msg.PayloadType,
				PayloadURL:  msg.PayloadURL,
				SubmittedAt: msg.SubmittedAt,
			},
			UserID:    &msg.UserID,
			UserEmail: msg.UserEmail,
		}

		if err := tx.Create(&savedSub).Error; err != nil {
			return err
		}
		created = true

		studentUserUuid = msg.UserUuid
		courseUuid = msg.CourseUuid
		contentTitle = msg.ContentTitle

		if studentUserUuid == uuid.Nil && msg.UserID != 0 {
			var u user.User
			if err := tx.Select("uuid").Where("id = ?", msg.UserID).First(&u).Error; err == nil {
				studentUserUuid = u.Uuid
			}
		}
		if (courseUuid == uuid.Nil || contentTitle == "") && msg.ContentID != 0 {
			var cnt content.Content
			if err := tx.Preload("Module.Course").Where("id = ?", msg.ContentID).First(&cnt).Error; err == nil {
				if contentTitle == "" {
					contentTitle = cnt.Title
				}
				if courseUuid == uuid.Nil && cnt.Module != nil && cnt.Module.Course != nil {
					courseUuid = cnt.Module.Course.Uuid
				}
			}
		}

		inter := content.StudentInteraction{
			UserID:      msg.UserID,
			UserEmail:   msg.UserEmail,
			ContentID:   msg.ContentID,
			CompletedAt: &msg.SubmittedAt,
		}
		if interErr := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "content_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"completed_at": gorm.Expr("COALESCE(student_interactions.completed_at, excluded.completed_at)"),
			}),
		}).Create(&inter).Error; interErr != nil {
			return fmt.Errorf("failed to record interaction: %w", interErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if created && s.notifications != nil && studentUserUuid != uuid.Nil {
		_, _ = s.notifications.NotifySystem(studentUserUuid, notification.NotificationContent{
			Title: "Assignment Saved",
			Body:  fmt.Sprintf("Your submission (attempt #%d) for %q has been saved.", savedSub.Attempt, contentTitle),
			URL:   fmt.Sprintf("/courses/%s", courseUuid),
		})
	}

	return &savedSub, nil
}

func (s *SubmissionService) NotifySubmitted(target *SubmissionTarget) {
	if s.notifications == nil || target == nil || target.UserUuid == uuid.Nil {
		return
	}
	_, err := s.notifications.NotifySystem(target.UserUuid, notification.NotificationContent{
		Title: "Assignment Submitted",
		Body:  fmt.Sprintf("Your submission for %q has been received and queued for processing.", target.ContentTitle),
		URL:   fmt.Sprintf("/courses/%s", target.CourseUuid),
	})
	if err != nil {
		s.logger.Errorf("Failed to send submission queued notification: %v", err)
	}
}

func (s *SubmissionService) NotifySubmissionFailed(userUuid, courseUuid uuid.UUID, contentTitle string) {
	if s.notifications == nil || userUuid == uuid.Nil {
		return
	}
	if contentTitle == "" {
		contentTitle = "Assignment"
	}
	url := "/courses"
	if courseUuid != uuid.Nil {
		url = fmt.Sprintf("/courses/%s", courseUuid)
	}
	_, err := s.notifications.NotifySystem(userUuid, notification.NotificationContent{
		Title: "Assignment Submission Failed",
		Body:  fmt.Sprintf("We could not process your submission for %q. Please try submitting again.", contentTitle),
		URL:   url,
	})
	if err != nil {
		s.logger.Errorf("Failed to send submission failed notification: %v", err)
	}
}
