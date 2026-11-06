package submission

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PayloadType identifies how a submission's deliverable is provided.
type PayloadType string

const (
	PayloadTypeURL  PayloadType = "URL"  // Link to a git repository or externally hosted document
	PayloadTypeFile PayloadType = "FILE" // Reserved for uploaded documents; not accepted yet
)

// Grade represents the evaluation score and instructor feedback for a submission.
type Grade struct {
	gorm.Model
	SubmissionID       uint        `json:"submissionId" gorm:"not null;uniqueIndex"`
	Submission         *Submission `json:"submission,omitempty" gorm:"foreignKey:SubmissionID"`
	Score              float64     `json:"score" gorm:"type:decimal(5,2);not null"`
	Feedback           string      `json:"feedback,omitempty" gorm:"type:text"`
	IsVisibleToStudent bool        `json:"isVisibleToStudent" gorm:"not null;default:false"`
	GradedByID         *uint       `json:"gradedById,omitempty" gorm:"index"`
	GradedBy           *user.User  `json:"gradedBy,omitempty" gorm:"foreignKey:GradedByID;constraint:OnDelete:SET NULL;"`
}

// Submission records student responses or deliverables for quizzes, coding exercises, and assignments.
type Submission struct {
	gorm.Model
	Uuid        uuid.UUID        `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	ContentID   uint             `json:"contentId" gorm:"not null;index"`
	Content     *content.Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Attempt     int              `json:"attempt" gorm:"not null;default:1"`
	IsLate      bool             `json:"isLate" gorm:"not null;default:false"`
	PayloadType PayloadType      `json:"payloadType" gorm:"type:varchar(20);not null;default:'URL'"`
	PayloadURL  string           `json:"payloadUrl,omitempty" gorm:"type:text"`
	FileName    string           `json:"fileName,omitempty" gorm:"type:varchar(255)"`
	FileSize    int64            `json:"fileSize,omitempty"`
	ContentType string           `json:"contentType,omitempty" gorm:"type:varchar(100)"`
	StorageKey  string           `json:"storageKey,omitempty" gorm:"type:varchar(512)"`
	SubmittedAt time.Time        `json:"submittedAt" gorm:"not null;index"`
	Grade       *Grade           `json:"grade,omitempty" gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE;"`
}

func (s *Submission) BeforeCreate(tx *gorm.DB) error {
	if s.Uuid == uuid.Nil {
		s.Uuid = uuid.New()
	}
	return nil
}

// GroupSubmission
type GroupSubmission struct {
	Submission
	GroupID *uint         `json:"groupId,omitempty" gorm:"index"`
	Group   *course.Group `json:"group,omitempty" gorm:"foreignKey:GroupID;constraint:OnDelete:SET NULL;"`
	UserID  *uint         `json:"userId,omitempty" gorm:"index"` // User that Submited
	User    *user.User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL;"`
}

// IndividualSubmission
type IndividualSubmission struct {
	Submission
	UserID    *uint      `json:"userId,omitempty" gorm:"index"`
	User      *user.User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL;"`
	UserEmail string     `json:"userEmail,omitempty" gorm:"type:varchar(255);index"`
}

type ReminderType string

const (
	ReminderOneDay     ReminderType = "24H"
	ReminderFifteenMin ReminderType = "15M"
)

// DeadlineReminder records that a deadline reminder has been sent to a student for an assignment.
type DeadlineReminder struct {
	ID           uint         `gorm:"primaryKey"`
	ContentID    uint         `gorm:"not null;uniqueIndex:idx_deadline_reminder_unique;index"`
	UserID       uint         `gorm:"not null;uniqueIndex:idx_deadline_reminder_unique;index"`
	ReminderType ReminderType `gorm:"type:varchar(10);not null;uniqueIndex:idx_deadline_reminder_unique"`
	SentAt       time.Time    `gorm:"not null"`
}

// UploadTicket reserves a submission slot and locks the submission receipt timestamp during two-phase file uploads.
type UploadTicket struct {
	gorm.Model
	Uuid        uuid.UUID `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	CourseID    uint      `json:"courseId" gorm:"not null;index"`
	ContentID   uint      `json:"contentId" gorm:"not null;index"`
	UserID      uint      `json:"userId" gorm:"not null;index"`
	UserEmail   string    `json:"userEmail" gorm:"type:varchar(255);not null"`
	SubmittedAt time.Time `json:"submittedAt" gorm:"not null"`
	ExpiresAt   time.Time `json:"expiresAt" gorm:"not null;index"`
	Consumed    bool      `json:"consumed" gorm:"not null;default:false"`
}

func (t *UploadTicket) BeforeCreate(tx *gorm.DB) error {
	if t.Uuid == uuid.Nil {
		t.Uuid = uuid.New()
	}
	return nil
}
