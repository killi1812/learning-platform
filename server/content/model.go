package content

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentType identifies the specific content category in the Class Table Inheritance hierarchy.
type ContentType string

const (
	TypeStaticPage ContentType = "STATIC_PAGE"
	TypeAssignment ContentType = "ASSIGNMENT"
)

// AssignmentType identifies the submittable assignment category (Class Table Inheritance below Assignment).
type AssignmentType string

const (
	AssignmentHandIn         AssignmentType = "HANDIN"          // Traditional hand-in (document / repository link)
	AssignmentQuiz           AssignmentType = "QUIZ"            // Interactive quiz, extended by Quiz
	AssignmentCodingExercise AssignmentType = "CODING_EXERCISE" // Automated coding task, extended by CodingExercise
)

// QuestionType defines the interaction format of a quiz question.
type QuestionType string

const (
	TypeMultipleChoice  QuestionType = "MULTIPLE_CHOICE"
	TypeOrdering        QuestionType = "ORDERING"
	TypeMatching        QuestionType = "MATCHING"
	TypeFillInTheBlanks QuestionType = "FILL_IN_THE_BLANKS"
)

// Content is the base entity for all instructional content items within a module (Class Table Inheritance).
type Content struct {
	gorm.Model
	Uuid         uuid.UUID      `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	ModuleID     uint           `json:"moduleId" gorm:"not null;index"`
	Module       *course.Module `json:"module,omitempty" gorm:"foreignKey:ModuleID;constraint:OnDelete:CASCADE;"`
	Title        string         `json:"title" gorm:"type:varchar(255);not null"`
	Position     int            `json:"position" gorm:"not null;default:0"`
	Type         ContentType    `json:"type" gorm:"type:varchar(50);not null"`
	VisibleFrom  *time.Time     `json:"visibleFrom,omitempty"`
	VisibleUntil *time.Time     `json:"visibleUntil,omitempty"`

	// 1-to-1 extension relations for Class Table Inheritance
	StaticPage *StaticPage `json:"staticPage,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Assignment *Assignment `json:"assignment,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`

	Interactions []StudentInteraction `json:"interactions,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
}

func (c *Content) BeforeCreate(tx *gorm.DB) error {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	return nil
}

// StaticPage represents course material embedded via iframe URL and/or formatted rich text.
type StaticPage struct {
	ContentID uint     `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content   *Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	SourceURL string   `json:"sourceUrl" gorm:"type:text;not null"` // Target URL rendered directly in client iframe
	RichText  string   `json:"richText,omitempty" gorm:"type:text"` // Optional text / description / markdown fallback
}

// Assignment is the base for every submittable content item (hand-ins, quizzes, coding exercises).
type Assignment struct {
	ContentID              uint           `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content                *Content       `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Type                   AssignmentType `json:"type" gorm:"type:varchar(50);not null;default:'HANDIN'"`
	Deadline               *time.Time     `json:"deadline,omitempty"`
	AllowsGroupSubmission  bool           `json:"allowsGroupSubmission" gorm:"not null;default:false"`
	AcceptsLateSubmissions bool           `json:"acceptsLateSubmissions" gorm:"not null;default:false"`

	// 1-to-1 extension relations for Class Table Inheritance (HANDIN has no subtype extension table)
	Quiz           *Quiz           `json:"quiz,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	CodingExercise *CodingExercise `json:"codingExercise,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
}

// Quiz is an Assignment subtype: an interactive knowledge check with configurable attempts and questions.
type Quiz struct {
	ContentID   uint        `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Assignment  *Assignment `json:"assignment,omitempty" gorm:"foreignKey:ContentID;references:ContentID;constraint:OnDelete:CASCADE;"`
	MaxAttempts int         `json:"maxAttempts" gorm:"not null;default:1"`
	Questions   []Question  `json:"questions,omitempty" gorm:"foreignKey:QuizID;references:ContentID;constraint:OnDelete:CASCADE;"`
}

// Question represents an individual quiz prompt supporting hints and automated grading criteria.
type Question struct {
	gorm.Model
	QuizID    uint         `json:"quizId" gorm:"not null;index"`
	Quiz      *Quiz        `json:"quiz,omitempty" gorm:"foreignKey:QuizID;references:ContentID;constraint:OnDelete:CASCADE;"`
	Position  int          `json:"position" gorm:"not null;default:0"`
	Type      QuestionType `json:"type" gorm:"type:varchar(50);not null"`
	Prompt    string       `json:"prompt" gorm:"type:text;not null"`
	Hint      string       `json:"hint,omitempty" gorm:"type:text"`
	Options   string       `json:"options,omitempty" gorm:"type:text"`   // JSON-encoded options / choices
	AnswerKey string       `json:"answerKey,omitempty" gorm:"type:text"` // Expected answer for automated grading
	Points    float64      `json:"points" gorm:"type:decimal(5,2);not null;default:1.0"`
}

// CodingExercise is an Assignment subtype: an automated coding task with starter code repo and test suites.
type CodingExercise struct {
	ContentID       uint        `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Assignment      *Assignment `json:"assignment,omitempty" gorm:"foreignKey:ContentID;references:ContentID;constraint:OnDelete:CASCADE;"`
	StarterCodeRepo string      `json:"starterCodeRepo" gorm:"type:varchar(255);not null"`
	// TODO: This needs more thinking it is not opnly string. But that is for later
	VisibleTestCases string `json:"visibleTestCases,omitempty" gorm:"type:text"`
	HiddenTestCases  string `json:"hiddenTestCases,omitempty" gorm:"type:text"`
}

// StudentInteraction tracks student engagement with specific content to evaluate prerequisite module completion.
type StudentInteraction struct {
	gorm.Model
	UserID      uint       `json:"userId" gorm:"not null;uniqueIndex:idx_user_content_interaction;index"`
	User        *user.User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	UserEmail   string     `json:"userEmail,omitempty" gorm:"type:varchar(255);index"`
	ContentID   uint       `json:"contentId" gorm:"not null;uniqueIndex:idx_user_content_interaction;index"`
	Content     *Content   `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
