package quiz

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"gorm.io/gorm"
)

// Attempt is one try by one student at one quiz. An attempt without SubmittedAt is in progress.
// A student has at most one in-progress attempt per quiz.
type Attempt struct {
	gorm.Model
	UserID      uint       `gorm:"not null;index:idx_attempt_user_quiz"`
	User        *user.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	QuizID      uint       `gorm:"not null;index:idx_attempt_user_quiz"` // content.Quiz.ContentID
	StartedAt   time.Time  `gorm:"not null"`
	SubmittedAt *time.Time
	Percent     float64  `gorm:"not null;default:0"`
	Passed      bool     `gorm:"not null;default:false"`
	Answers     []Answer `gorm:"foreignKey:AttemptID;constraint:OnDelete:CASCADE;"`
}

// Answer stores what a student marked for one question. Selected is a JSON array of option indices.
// Correctness is never stored here: grading happens only when the attempt is submitted.
type Answer struct {
	gorm.Model
	AttemptID  uint              `gorm:"not null;uniqueIndex:idx_answer_attempt_question"`
	QuestionID uint              `gorm:"not null;uniqueIndex:idx_answer_attempt_question"`
	Question   *content.Question `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE;"`
	Selected   string            `gorm:"type:text;not null"`
}
