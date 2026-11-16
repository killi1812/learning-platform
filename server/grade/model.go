package grade

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"gorm.io/gorm"
)

// Grade represents the evaluation score and instructor feedback for a submission.
type Grade struct {
	gorm.Model
	IndividualSubmissionID *uint                            `json:"individualSubmissionId,omitempty" gorm:"uniqueIndex"`
	IndividualSubmission   *submission.IndividualSubmission `json:"individualSubmission,omitempty" gorm:"foreignKey:IndividualSubmissionID;constraint:OnDelete:CASCADE;"`
	GroupSubmissionID      *uint                            `json:"groupSubmissionId,omitempty" gorm:"uniqueIndex"`
	GroupSubmission        *submission.GroupSubmission      `json:"groupSubmission,omitempty" gorm:"foreignKey:GroupSubmissionID;constraint:OnDelete:CASCADE;"`
	Score                  float64                          `json:"score" gorm:"type:decimal(5,2);not null"`
	Feedback               string                           `json:"feedback,omitempty" gorm:"type:text"`
	IsVisibleToStudent     bool                             `json:"isVisibleToStudent" gorm:"not null;default:false"`
	ReleasedAt             *time.Time                       `json:"releasedAt,omitempty"`
	GradedByID             *uint                            `json:"gradedById,omitempty" gorm:"index"`
	GradedBy               *user.User                       `json:"gradedBy,omitempty" gorm:"foreignKey:GradedByID;constraint:OnDelete:SET NULL;"`
}
