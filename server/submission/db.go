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
	ErrTicketNotFound            = errors.New("upload ticket not found")
	ErrTicketExpired             = errors.New("upload ticket has expired")
	ErrTicketConsumed            = errors.New("upload ticket has already been used")
	ErrSubmissionNotFound        = errors.New("submission not found")
	ErrUnauthorizedSubmission    = errors.New("unauthorized access to submission")
)

// SubmissionTarget contains database primary IDs for publishing the submission message.
type SubmissionTarget struct {
	CourseID     uint
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
	InitFileSubmission(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*UploadTicket, error)
	ValidateAndExtendTicket(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*UploadTicket, error)
	ConsumeUploadTicket(ticketUuid uuid.UUID) error
	SaveFromMessage(msg SubmissionMessage) (*IndividualSubmission, error)
	NotifySubmitted(target *SubmissionTarget)
	NotifySubmissionFailed(userUuid, courseUuid uuid.UUID, contentTitle string)
	CheckDeadlineReminders(now time.Time) (int, error)
	ListMySubmissions(courseUuid, contentUuid, userUuid uuid.UUID) ([]SubmissionDto, error)
	ListCourseAssignmentSubmissions(courseUuid, contentUuid uuid.UUID) ([]SubmissionDto, error)
	GetSubmissionForDownload(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*IndividualSubmission, error)
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
		CourseID:     enrollment.CourseID,
		ContentID:    cnt.ID,
		ContentUuid:  cnt.Uuid,
		ContentTitle: cnt.Title,
		CourseUuid:   courseUuid,
		UserID:       u.ID,
		UserUuid:     u.Uuid,
		UserEmail:    u.Email,
	}, nil
}

// InitFileSubmission reserves an upload ticket and stamps the receipt time.
func (s *SubmissionService) InitFileSubmission(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*UploadTicket, error) {
	target, err := s.ResolveTarget(courseUuid, contentUuid, userUuid, submittedAt)
	if err != nil {
		return nil, err
	}

	// Invalidate any existing unconsumed tickets for this student and assignment
	_ = s.db.Model(&UploadTicket{}).
		Where("user_id = ? AND content_id = ? AND consumed = ?", target.UserID, target.ContentID, false).
		Update("consumed", true).Error

	ticket := UploadTicket{
		Uuid:         uuid.New(),
		CourseID:     target.CourseID,
		ContentID:    target.ContentID,
		ContentTitle: target.ContentTitle,
		UserID:       target.UserID,
		UserEmail:    target.UserEmail,
		SubmittedAt:  submittedAt,
		ExpiresAt:    submittedAt.Add(30 * time.Minute),
		Consumed:     false,
	}

	if err := s.db.Create(&ticket).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

// ValidateAndExtendTicket verifies that the ticket is valid, belongs to the student, and extends its expiration while active.
func (s *SubmissionService) ValidateAndExtendTicket(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*UploadTicket, error) {
	var ticket UploadTicket
	err := s.db.
		Joins("JOIN contents ON contents.id = upload_tickets.content_id").
		Joins("JOIN courses ON courses.id = upload_tickets.course_id").
		Joins("JOIN users ON users.id = upload_tickets.user_id").
		Where("upload_tickets.uuid = ? AND courses.uuid = ? AND contents.uuid = ? AND users.uuid = ?", ticketUuid, courseUuid, contentUuid, userUuid).
		First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}

	if ticket.Consumed {
		return nil, ErrTicketConsumed
	}

	if now.After(ticket.ExpiresAt) {
		return nil, ErrTicketExpired
	}

	// Extend ticket expiration by 15 minutes during active transfer so ongoing slow uploads stay valid
	if ticket.ExpiresAt.Sub(now) < 15*time.Minute {
		ticket.ExpiresAt = now.Add(15 * time.Minute)
		_ = s.db.Model(&ticket).Update("expires_at", ticket.ExpiresAt).Error
	}

	return &ticket, nil
}

// ConsumeUploadTicket marks the upload ticket as consumed atomically.
func (s *SubmissionService) ConsumeUploadTicket(ticketUuid uuid.UUID) error {
	res := s.db.Model(&UploadTicket{}).Where("uuid = ? AND consumed = ?", ticketUuid, false).Update("consumed", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var exists int64
		if err := s.db.Model(&UploadTicket{}).Where("uuid = ?", ticketUuid).Count(&exists).Error; err == nil && exists == 0 {
			return ErrTicketNotFound
		}
		return ErrTicketConsumed
	}
	return nil
}

// SaveFromMessage persists an enqueued submission into the database. Idempotent on msg.Uuid.
func (s *SubmissionService) SaveFromMessage(msg SubmissionMessage) (*IndividualSubmission, error) {
	// Fast check for duplicate delivery
	var existing IndividualSubmission
	if err := s.db.Where("uuid = ?", msg.Uuid).Limit(1).Find(&existing).Error; err == nil && existing.ID != 0 {
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
		if err := tx.Where("uuid = ?", msg.Uuid).Limit(1).Find(&savedSub).Error; err == nil && savedSub.ID != 0 {
			return nil
		}

		// Fetch assignment to compute isLate
		var asgn content.Assignment
		if err := tx.Where("content_id = ?", msg.ContentID).First(&asgn).Error; err != nil {
			return fmt.Errorf("assignment for content %d not found: %w", msg.ContentID, err)
		}

		// Compute attempt number for this student on this assignment
		var lastSub IndividualSubmission
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("content_id = ? AND user_id = ?", msg.ContentID, msg.UserID).
			Order("attempt DESC").
			Limit(1).
			Find(&lastSub).Error
		if err != nil {
			return fmt.Errorf("failed to query latest attempt: %w", err)
		}

		maxAttempt := 0
		if lastSub.ID != 0 {
			maxAttempt = lastSub.Attempt
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
				FileName:    msg.FileName,
				FileSize:    msg.FileSize,
				ContentType: msg.ContentType,
				StorageKey:  msg.StorageKey,
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

	if created && studentUserUuid != uuid.Nil {
		_, _ = s.notifications.NotifySystem(studentUserUuid, notification.NotificationContent{
			Title: "Assignment Saved",
			Body:  fmt.Sprintf("Your submission (attempt #%d) for %q has been saved.", savedSub.Attempt, contentTitle),
			URL:   fmt.Sprintf("/courses/%s", courseUuid),
		})
	}

	return &savedSub, nil
}

func (s *SubmissionService) NotifySubmitted(target *SubmissionTarget) {
	if target == nil || target.UserUuid == uuid.Nil {
		s.logger.Error("Cannot notify submission queued: target or user uuid is nil")
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
	if userUuid == uuid.Nil {
		s.logger.Error("Cannot notify submission failed: user uuid is nil")
		return
	}
	targetURL := "/dashboard"
	if courseUuid != uuid.Nil {
		targetURL = fmt.Sprintf("/courses/%s", courseUuid)
	}
	_, err := s.notifications.NotifySystem(userUuid, notification.NotificationContent{
		Title: "Assignment Submission Failed",
		Body:  fmt.Sprintf("We could not process your submission for %q. Please contact your professor so they can help you resolve it.", contentTitle),
		URL:   targetURL,
	})
	if err != nil {
		s.logger.Errorf("Failed to send submission failed notification: %v", err)
	}
}

// ListMySubmissions returns all submissions for a student on a specific assignment.
func (s *SubmissionService) ListMySubmissions(courseUuid, contentUuid, userUuid uuid.UUID) ([]SubmissionDto, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotEnrolled
		}
		return nil, err
	}

	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	var subs []IndividualSubmission
	err = s.db.
		Where("content_id = ? AND user_id = ?", cnt.ID, u.ID).
		Order("attempt DESC").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}

	dtos := make([]SubmissionDto, 0, len(subs))
	for _, sub := range subs {
		dtos = append(dtos, SubmissionDto{
			Uuid:        sub.Uuid,
			Attempt:     sub.Attempt,
			IsLate:      sub.IsLate,
			PayloadType: sub.PayloadType,
			PayloadURL:  sub.PayloadURL,
			FileName:    sub.FileName,
			FileSize:    sub.FileSize,
			ContentType: sub.ContentType,
			SubmittedAt: sub.SubmittedAt,
			UserEmail:   sub.UserEmail,
		})
	}
	return dtos, nil
}

// ListCourseAssignmentSubmissions returns all submissions for an assignment (for instructors).
func (s *SubmissionService) ListCourseAssignmentSubmissions(courseUuid, contentUuid uuid.UUID) ([]SubmissionDto, error) {
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	var subs []IndividualSubmission
	err = s.db.
		Where("content_id = ?", cnt.ID).
		Order("submitted_at DESC").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}

	dtos := make([]SubmissionDto, 0, len(subs))
	for _, sub := range subs {
		dtos = append(dtos, SubmissionDto{
			Uuid:        sub.Uuid,
			Attempt:     sub.Attempt,
			IsLate:      sub.IsLate,
			PayloadType: sub.PayloadType,
			PayloadURL:  sub.PayloadURL,
			FileName:    sub.FileName,
			FileSize:    sub.FileSize,
			ContentType: sub.ContentType,
			SubmittedAt: sub.SubmittedAt,
			UserEmail:   sub.UserEmail,
		})
	}
	return dtos, nil
}

// GetSubmissionForDownload retrieves a submission and verifies download access authorization.
func (s *SubmissionService) GetSubmissionForDownload(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*IndividualSubmission, error) {
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	var sub IndividualSubmission
	err = s.db.
		Where("uuid = ? AND content_id = ?", subUuid, cnt.ID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	if !isPrivileged {
		var u user.User
		if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
			return nil, ErrUnauthorizedSubmission
		}
		if sub.UserID == nil || *sub.UserID != u.ID {
			return nil, ErrUnauthorizedSubmission
		}
	}

	return &sub, nil
}
