package submission

import (
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"gorm.io/gorm"
)

// Grade represents the evaluation score and instructor feedback for a submission.
type Grade struct {
	gorm.Model
	SubmissionID       uint        `json:"submissionId" gorm:"not null;uniqueIndex"`
	Submission         *Submission `json:"submission,omitempty" gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE;"`
	Score              float64     `json:"score" gorm:"type:decimal(5,2);not null"`
	Feedback           string      `json:"feedback,omitempty" gorm:"type:text"`
	IsVisibleToStudent bool        `json:"isVisibleToStudent" gorm:"not null;default:false"`
	GradedByID         *uint       `json:"gradedById,omitempty" gorm:"index"`
	GradedBy           *user.User  `json:"gradedBy,omitempty" gorm:"foreignKey:GradedByID;constraint:OnDelete:SET NULL;"`
}

// Submission records student responses or deliverables for quizzes, coding exercises, and assignments.
type Submission struct {
	gorm.Model
	ContentID  uint             `json:"contentId" gorm:"not null;index"`
	Content    *content.Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Attempt    int              `json:"attempt" gorm:"not null;default:1"`
	IsLate     bool             `json:"isLate" gorm:"not null;default:false"`
	PayloadURL string           `json:"payloadUrl" gorm:"type:text;not null"`
	Grade      *Grade           `json:"grade,omitempty" gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE;"`
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
