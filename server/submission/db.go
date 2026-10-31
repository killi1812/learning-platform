package submission

import (
	"errors"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	ContentID uint
	UserID    uint
	UserEmail string
}

// ISubmissionService provides domain validation and database persistence for submissions.
type ISubmissionService interface {
	ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*SubmissionTarget, error)
	SaveFromMessage(msg SubmissionMessage) (*IndividualSubmission, error)
}

// SubmissionService implements ISubmissionService using GORM.
type SubmissionService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewSubmissionService constructs a new SubmissionService.
func NewSubmissionService(db *gorm.DB, logger *zap.SugaredLogger) ISubmissionService {
	return &SubmissionService{
		db:     db,
		logger: logger,
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
		ContentID: cnt.ID,
		UserID:    u.ID,
		UserEmail: u.Email,
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

		var inter content.StudentInteraction
		interErr := tx.Where("user_id = ? AND content_id = ?", msg.UserID, msg.ContentID).First(&inter).Error
		if errors.Is(interErr, gorm.ErrRecordNotFound) {
			inter = content.StudentInteraction{
				UserID:      msg.UserID,
				UserEmail:   msg.UserEmail,
				ContentID:   msg.ContentID,
				CompletedAt: &msg.SubmittedAt,
			}
			if err := tx.Create(&inter).Error; err != nil {
				return fmt.Errorf("failed to record interaction: %w", err)
			}
		} else if interErr == nil && inter.CompletedAt == nil {
			inter.CompletedAt = &msg.SubmittedAt
			if err := tx.Save(&inter).Error; err != nil {
				return fmt.Errorf("failed to update interaction: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &savedSub, nil
}
