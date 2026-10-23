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
	TypeStaticPage     ContentType = "STATIC_PAGE"
	TypeQuiz           ContentType = "QUIZ"
	TypeAssignment     ContentType = "ASSIGNMENT"
	TypeCodingExercise ContentType = "CODING_EXERCISE"
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
	Uuid         uuid.UUID      `json:"uuid" gorm:"type:uuid;uniqueIndex"`
	ModuleID     uint           `json:"moduleId" gorm:"not null;index"`
	Module       *course.Module `json:"module,omitempty" gorm:"foreignKey:ModuleID;constraint:OnDelete:CASCADE;"`
	Title        string         `json:"title" gorm:"type:varchar(255);not null"`
	Position     int            `json:"position" gorm:"not null;default:0"`
	Type         ContentType    `json:"type" gorm:"type:varchar(50);not null"`
	VisibleFrom  *time.Time     `json:"visibleFrom,omitempty"`
	VisibleUntil *time.Time     `json:"visibleUntil,omitempty"`

	// 1-to-1 extension relations for Class Table Inheritance
	StaticPage     *StaticPage     `json:"staticPage,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Quiz           *Quiz           `json:"quiz,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Assignment     *Assignment     `json:"assignment,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	CodingExercise *CodingExercise `json:"codingExercise,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`

	Interactions []StudentInteraction `json:"interactions,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
}

func (c *Content) BeforeCreate(tx *gorm.DB) error {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	return nil
}

// StaticPage represents course material (e.g. .pdf, .pptx, .md, .txt) stored in object storage or linked.
type StaticPage struct {
	ContentID   uint     `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content     *Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	FileName    string   `json:"fileName,omitempty" gorm:"type:varchar(255)"`
	FileSize    int64    `json:"fileSize,omitempty"`
	ContentType string   `json:"contentType,omitempty" gorm:"type:varchar(100)"`
	StorageKey  string   `json:"storageKey,omitempty" gorm:"type:varchar(255)"`
	SourceURL   string   `json:"sourceUrl,omitempty" gorm:"type:text"`
	RichText    string   `json:"richText,omitempty" gorm:"type:text"`
}

// Quiz represents an interactive knowledge check with configurable attempts, deadline, and questions.
type Quiz struct {
	ContentID   uint       `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content     *Content   `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	MaxAttempts int        `json:"maxAttempts" gorm:"not null;default:1"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Questions   []Question `json:"questions,omitempty" gorm:"foreignKey:QuizID;references:ContentID;constraint:OnDelete:CASCADE;"`
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

// Assignment represents a deliverable hand-in task (e.g. document uploads or reports).
type Assignment struct {
	ContentID              uint       `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content                *Content   `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission" gorm:"not null;default:false"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions" gorm:"not null;default:false"`
}

// CodingExercise represents an automated coding task with starter code repo, test suites, and deadline.
type CodingExercise struct {
	ContentID              uint       `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content                *Content   `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	StarterCodeRepo        string     `json:"starterCodeRepo" gorm:"type:varchar(255);not null"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission" gorm:"not null;default:false"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions" gorm:"not null;default:false"`
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
