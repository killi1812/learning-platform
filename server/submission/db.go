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
	ErrNoGroup                   = errors.New("you must be in a group to submit this assignment")
	ErrGroupChanged              = errors.New("group membership has changed since upload was initiated")
)

const (
	initialTicketExpiry  = 30 * time.Minute
	extendTicketDuration = 15 * time.Minute
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
	GroupID      *uint
	GroupName    string
	Members      []GroupMember
}

// ISubmissionService provides domain validation and database persistence for submissions.
type ISubmissionService interface {
	ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*SubmissionTarget, error)
	InitFileSubmission(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*UploadTicket, error)
	ValidateAndExtendTicket(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*UploadTicket, error)
	ResolveGroupMembers(groupID uint) (string, []GroupMember, error)
	ConsumeUploadTicket(ticketUuid uuid.UUID) error
	SaveFromMessage(msg SubmissionMessage) (*Submission, error)
	NotifySubmitted(target *SubmissionTarget)
	NotifySubmissionFailed(userUuid, courseUuid uuid.UUID, contentTitle string)
	CheckDeadlineReminders(now time.Time) (int, error)
	ListMySubmissions(courseUuid, contentUuid, userUuid uuid.UUID) ([]SubmissionDto, error)
	ListCourseAssignmentSubmissions(courseUuid, contentUuid uuid.UUID) ([]SubmissionDto, error)
	GetSubmissionForDownload(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*Submission, error)
}

// SubmissionService implements ISubmissionService using GORM.
type SubmissionService struct {
	db            *gorm.DB
	notifications notification.INotifier
	logger        *zap.SugaredLogger
}

// NewSubmissionService constructs a new SubmissionService.
func NewSubmissionService(db *gorm.DB, notifications notification.INotifier, logger *zap.SugaredLogger) ISubmissionService {
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

	target := &SubmissionTarget{
		CourseID:     enrollment.CourseID,
		ContentID:    cnt.ID,
		ContentUuid:  cnt.Uuid,
		ContentTitle: cnt.Title,
		CourseUuid:   courseUuid,
		UserID:       u.ID,
		UserUuid:     u.Uuid,
		UserEmail:    u.Email,
	}

	if cnt.Assignment.AllowsGroupSubmission {
		var grp course.Group
		err = s.db.
			Joins("JOIN group_members ON group_members.group_id = groups.id").
			Preload("Members").
			Where("groups.course_id = ? AND group_members.user_id = ? AND groups.deleted_at IS NULL", enrollment.CourseID, u.ID).
			First(&grp).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNoGroup
			}
			return nil, err
		}

		target.GroupID = &grp.ID
		target.GroupName = grp.Name
		members := make([]GroupMember, 0, len(grp.Members))
		for _, m := range grp.Members {
			members = append(members, GroupMember{
				UserID: m.ID,
				Uuid:   m.Uuid,
				Email:  m.Email,
			})
		}
		target.Members = members
	}

	return target, nil
}

// InitFileSubmission reserves an upload ticket and stamps the receipt time.
func (s *SubmissionService) InitFileSubmission(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*UploadTicket, error) {
	target, err := s.ResolveTarget(courseUuid, contentUuid, userUuid, submittedAt)
	if err != nil {
		return nil, err
	}

	if target.GroupID != nil {
		// Invalidate any existing unconsumed tickets for this entire group and assignment
		_ = s.db.Model(&UploadTicket{}).
			Where("group_id = ? AND content_id = ? AND consumed = ?", *target.GroupID, target.ContentID, false).
			Update("consumed", true).Error
	} else {
		// Invalidate any existing unconsumed tickets for this student and assignment
		_ = s.db.Model(&UploadTicket{}).
			Where("user_id = ? AND content_id = ? AND consumed = ?", target.UserID, target.ContentID, false).
			Update("consumed", true).Error
	}

	ticket := UploadTicket{
		Uuid:         uuid.New(),
		CourseID:     target.CourseID,
		ContentID:    target.ContentID,
		ContentTitle: target.ContentTitle,
		UserID:       target.UserID,
		UserEmail:    target.UserEmail,
		GroupID:      target.GroupID,
		SubmittedAt:  submittedAt,
		ExpiresAt:    submittedAt.Add(initialTicketExpiry),
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

	if ticket.GroupID != nil {
		var memberCount int64
		if err := s.db.Table("group_members").
			Where("group_id = ? AND user_id = ?", *ticket.GroupID, ticket.UserID).
			Count(&memberCount).Error; err != nil {
			return nil, err
		}
		if memberCount == 0 {
			return nil, ErrGroupChanged
		}
	}

	// Extend ticket expiration by 15 minutes during active transfer so ongoing slow uploads stay valid
	if ticket.ExpiresAt.Sub(now) < extendTicketDuration {
		ticket.ExpiresAt = now.Add(extendTicketDuration)
		_ = s.db.Model(&ticket).Update("expires_at", ticket.ExpiresAt).Error
	}

	return &ticket, nil
}

// ResolveGroupMembers retrieves the latest active group members snapshot for a group.
func (s *SubmissionService) ResolveGroupMembers(groupID uint) (string, []GroupMember, error) {
	var grp course.Group
	if err := s.db.Preload("Members").Where("id = ? AND deleted_at IS NULL", groupID).First(&grp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrNoGroup
		}
		return "", nil, err
	}
	members := make([]GroupMember, 0, len(grp.Members))
	for _, m := range grp.Members {
		members = append(members, GroupMember{
			UserID: m.ID,
			Uuid:   m.Uuid,
			Email:  m.Email,
		})
	}
	return grp.Name, members, nil
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
func (s *SubmissionService) SaveFromMessage(msg SubmissionMessage) (*Submission, error) {
	// Fast check for duplicate delivery in both individual and group submissions
	var existingInd IndividualSubmission
	if err := s.db.Where("uuid = ?", msg.Uuid).Limit(1).Find(&existingInd).Error; err == nil && existingInd.ID != 0 {
		s.logger.Debugf("Submission %s already exists in individual_submissions, skipping duplicate message", msg.Uuid)
		return &existingInd.Submission, nil
	}
	var existingGrp GroupSubmission
	if err := s.db.Where("uuid = ?", msg.Uuid).Limit(1).Find(&existingGrp).Error; err == nil && existingGrp.ID != 0 {
		s.logger.Debugf("Submission %s already exists in group_submissions, skipping duplicate message", msg.Uuid)
		return &existingGrp.Submission, nil
	}

	var savedSub Submission
	var created bool
	var courseUuid uuid.UUID
	var contentTitle string
	var studentUserUuid uuid.UUID
	var isGroup bool
	var groupMemberUuids []uuid.UUID

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Re-check within transaction
		var recheckInd IndividualSubmission
		if err := tx.Where("uuid = ?", msg.Uuid).Limit(1).Find(&recheckInd).Error; err == nil && recheckInd.ID != 0 {
			savedSub = recheckInd.Submission
			return nil
		}
		var recheckGrp GroupSubmission
		if err := tx.Where("uuid = ?", msg.Uuid).Limit(1).Find(&recheckGrp).Error; err == nil && recheckGrp.ID != 0 {
			savedSub = recheckGrp.Submission
			return nil
		}

		// Fetch assignment to compute isLate
		var asgn content.Assignment
		if err := tx.Where("content_id = ?", msg.ContentID).First(&asgn).Error; err != nil {
			return fmt.Errorf("assignment for content %d not found: %w", msg.ContentID, err)
		}

		isLate := asgn.Deadline != nil && msg.SubmittedAt.After(*asgn.Deadline)

		if msg.GroupID != nil {
			isGroup = true
			var lastSub GroupSubmission
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("content_id = ? AND group_id = ?", msg.ContentID, *msg.GroupID).
				Order("attempt DESC").
				Limit(1).
				Find(&lastSub).Error
			if err != nil {
				return fmt.Errorf("failed to query latest group attempt: %w", err)
			}

			maxAttempt := 0
			if lastSub.ID != 0 {
				maxAttempt = lastSub.Attempt
			}

			savedGrp := GroupSubmission{
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
				GroupID:   msg.GroupID,
				UserID:    &msg.UserID,
				UserEmail: msg.UserEmail,
			}

			if err := tx.Create(&savedGrp).Error; err != nil {
				return err
			}

			if len(msg.Members) > 0 {
				var memberUsers []user.User
				memberIDs := make([]uint, 0, len(msg.Members))
				for _, m := range msg.Members {
					memberIDs = append(memberIDs, m.UserID)
					if m.Uuid != uuid.Nil {
						groupMemberUuids = append(groupMemberUuids, m.Uuid)
					}
				}
				if err := tx.Where("id IN ?", memberIDs).Find(&memberUsers).Error; err != nil {
					return fmt.Errorf("failed to load group members: %w", err)
				}
				if err := tx.Model(&savedGrp).Association("Members").Append(&memberUsers); err != nil {
					return fmt.Errorf("failed to save group submission members: %w", err)
				}
			}

			savedSub = savedGrp.Submission
			created = true

			// Record interactions for all group members in snapshot
			for _, m := range msg.Members {
				inter := content.StudentInteraction{
					UserID:      m.UserID,
					UserEmail:   m.Email,
					ContentID:   msg.ContentID,
					CompletedAt: &msg.SubmittedAt,
				}
				if interErr := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "user_id"}, {Name: "content_id"}},
					DoUpdates: clause.Assignments(map[string]any{
						"completed_at": gorm.Expr("COALESCE(student_interactions.completed_at, excluded.completed_at)"),
					}),
				}).Create(&inter).Error; interErr != nil {
					return fmt.Errorf("failed to record group member interaction: %w", interErr)
				}
			}
		} else {
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

			savedInd := IndividualSubmission{
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

			if err := tx.Create(&savedInd).Error; err != nil {
				return err
			}
			savedSub = savedInd.Submission
			created = true

			studentUserUuid = msg.UserUuid
			if studentUserUuid == uuid.Nil && msg.UserID != 0 {
				var u user.User
				if err := tx.Select("uuid").Where("id = ?", msg.UserID).First(&u).Error; err == nil {
					studentUserUuid = u.Uuid
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
		}

		courseUuid = msg.CourseUuid
		contentTitle = msg.ContentTitle
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

		return nil
	})
	if err != nil {
		return nil, err
	}

	if created {
		if isGroup {
			if len(groupMemberUuids) > 0 {
				_, _ = s.notifications.NotifySystemMany(groupMemberUuids, notification.NotificationContent{
					Title: "Assignment Saved",
					Body:  fmt.Sprintf("%s submitted attempt #%d for %q on behalf of your group.", msg.UserEmail, savedSub.Attempt, contentTitle),
					URL:   fmt.Sprintf("/courses/%s", courseUuid),
				})
			}
		} else if studentUserUuid != uuid.Nil {
			_, _ = s.notifications.NotifySystem(studentUserUuid, notification.NotificationContent{
				Title: "Assignment Saved",
				Body:  fmt.Sprintf("Your submission (attempt #%d) for %q has been saved.", savedSub.Attempt, contentTitle),
				URL:   fmt.Sprintf("/courses/%s", courseUuid),
			})
		}
	}

	return &savedSub, nil
}

func (s *SubmissionService) NotifySubmitted(target *SubmissionTarget) {
	if target == nil {
		s.logger.Error("Cannot notify submission queued: target is nil")
		return
	}

	if target.GroupID != nil && len(target.Members) > 0 {
		memberUuids := make([]uuid.UUID, 0, len(target.Members))
		for _, m := range target.Members {
			if m.Uuid != uuid.Nil {
				memberUuids = append(memberUuids, m.Uuid)
			}
		}
		if len(memberUuids) > 0 {
			_, err := s.notifications.NotifySystemMany(memberUuids, notification.NotificationContent{
				Title: "Assignment Submitted",
				Body:  fmt.Sprintf("A submission for %q on behalf of %s has been received and queued for processing.", target.ContentTitle, target.GroupName),
				URL:   fmt.Sprintf("/courses/%s", target.CourseUuid),
			})
			if err != nil {
				s.logger.Errorf("Failed to send group submission queued notifications: %v", err)
			}
			return
		}
	}

	if target.UserUuid == uuid.Nil {
		s.logger.Error("Cannot notify submission queued: user uuid is nil")
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

func gradeToDto(g *Grade) *SubmissionGradeDto {
	if g == nil {
		return nil
	}
	return &SubmissionGradeDto{
		Score:              g.Score,
		Feedback:           g.Feedback,
		IsVisibleToStudent: g.IsVisibleToStudent,
		GradedAt:           g.UpdatedAt,
		ReleasedAt:         g.ReleasedAt,
	}
}

func groupSubmissionsToDtos(grpSubs []GroupSubmission) []SubmissionDto {
	dtos := make([]SubmissionDto, 0, len(grpSubs))
	for _, sub := range grpSubs {
		var memberEmails []string
		for _, m := range sub.Members {
			memberEmails = append(memberEmails, m.Email)
		}
		groupName := ""
		if sub.Group != nil {
			groupName = sub.Group.Name
		}
		dtos = append(dtos, SubmissionDto{
			Uuid:         sub.Uuid,
			Attempt:      sub.Attempt,
			IsLate:       sub.IsLate,
			PayloadType:  sub.PayloadType,
			PayloadURL:   sub.PayloadURL,
			FileName:     sub.FileName,
			FileSize:     sub.FileSize,
			ContentType:  sub.ContentType,
			SubmittedAt:  sub.SubmittedAt,
			UserEmail:    sub.UserEmail,
			IsGroup:      true,
			GroupName:    groupName,
			MemberEmails: memberEmails,
			Grade:        gradeToDto(sub.Grade),
		})
	}
	return dtos
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
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	if cnt.Assignment != nil && cnt.Assignment.AllowsGroupSubmission {
		var grpSubs []GroupSubmission
		err = s.db.
			Preload("Group", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
			Preload("Members").
			Preload("Grade", "is_visible_to_student = ?", true).
			Where("content_id = ? AND (user_id = ? OR id IN (SELECT group_submission_id FROM group_submission_members WHERE user_id = ?))", cnt.ID, u.ID, u.ID).
			Order("attempt DESC").
			Find(&grpSubs).Error
		if err != nil {
			return nil, err
		}
		return groupSubmissionsToDtos(grpSubs), nil
	}

	var subs []IndividualSubmission
	err = s.db.
		Preload("Grade", "is_visible_to_student = ?", true).
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
			Grade:       gradeToDto(sub.Grade),
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
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	if cnt.Assignment != nil && cnt.Assignment.AllowsGroupSubmission {
		var grpSubs []GroupSubmission
		err = s.db.
			Preload("Group", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
			Preload("Members").
			Preload("Grade").
			Where("content_id = ?", cnt.ID).
			Order("submitted_at DESC").
			Find(&grpSubs).Error
		if err != nil {
			return nil, err
		}
		return groupSubmissionsToDtos(grpSubs), nil
	}

	var subs []IndividualSubmission
	err = s.db.
		Preload("Grade").
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
			Grade:       gradeToDto(sub.Grade),
		})
	}
	return dtos, nil
}

// GetSubmissionForDownload retrieves a submission and verifies download access authorization.
func (s *SubmissionService) GetSubmissionForDownload(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*Submission, error) {
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

	var indSub IndividualSubmission
	err = s.db.Where("uuid = ? AND content_id = ?", subUuid, cnt.ID).First(&indSub).Error
	if err == nil {
		if !isPrivileged {
			var u user.User
			if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
				return nil, ErrUnauthorizedSubmission
			}
			if indSub.UserID == nil || *indSub.UserID != u.ID {
				return nil, ErrUnauthorizedSubmission
			}
		}
		return &indSub.Submission, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var grpSub GroupSubmission
	err = s.db.Preload("Members").Where("uuid = ? AND content_id = ?", subUuid, cnt.ID).First(&grpSub).Error
	if err == nil {
		if !isPrivileged {
			var u user.User
			if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
				return nil, ErrUnauthorizedSubmission
			}
			allowed := (grpSub.UserID != nil && *grpSub.UserID == u.ID)
			if !allowed {
				for _, m := range grpSub.Members {
					if m.ID == u.ID {
						allowed = true
						break
					}
				}
			}
			if !allowed {
				return nil, ErrUnauthorizedSubmission
			}
		}
		return &grpSub.Submission, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubmissionNotFound
	}
	return nil, err
}
