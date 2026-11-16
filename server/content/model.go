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
	TypeFile       ContentType = "FILE"
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
	TypeTrueFalse       QuestionType = "TRUE_FALSE"
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

	File         *File                `json:"file,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Assignment   *Assignment          `json:"assignment,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Interactions []StudentInteraction `json:"interactions,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
}

func (c *Content) BeforeCreate(tx *gorm.DB) error {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	return nil
}

// File represents course material (e.g. .pdf, .pptx, .md, .txt) stored in object storage or linked.
type File struct {
	ContentID   uint     `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content     *Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	FileName    string   `json:"fileName,omitempty" gorm:"type:varchar(255)"`
	FileSize    int64    `json:"fileSize,omitempty"`
	ContentType string   `json:"contentType,omitempty" gorm:"type:varchar(100)"`
	StorageKey  string   `json:"storageKey,omitempty" gorm:"type:varchar(512)"`
	SourceURL   string   `json:"sourceUrl,omitempty" gorm:"type:text"`
	RichText    string   `json:"richText,omitempty" gorm:"type:text"`
}

// DefaultHandinMaxPoints is the default maximum score for a hand-in assignment.
const DefaultHandinMaxPoints = 5.0

// Assignment is the base for every submittable content item (hand-ins, quizzes, coding exercises).
type Assignment struct {
	ContentID              uint           `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	Content                *Content       `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE;"`
	Type                   AssignmentType `json:"type" gorm:"type:varchar(50);not null;default:'HANDIN'"`
	Deadline               *time.Time     `json:"deadline,omitempty"`
	AllowsGroupSubmission  bool           `json:"allowsGroupSubmission" gorm:"not null;default:false"`
	AcceptsLateSubmissions bool           `json:"acceptsLateSubmissions" gorm:"not null;default:false"`
	MaxPoints              float64        `json:"maxPoints" gorm:"type:decimal(5,2);not null;default:5"`

	// 1-to-1 extension relations for Class Table Inheritance (HANDIN has no subtype extension table)
	Quiz           *Quiz           `json:"quiz,omitempty" gorm:"foreignKey:ContentID;references:ContentID;constraint:OnDelete:CASCADE;"`
	CodingExercise *CodingExercise `json:"codingExercise,omitempty" gorm:"foreignKey:ContentID;references:ContentID;constraint:OnDelete:CASCADE;"`
}

// Quiz is an Assignment subtype: an interactive knowledge check with configurable attempts and questions.
type Quiz struct {
	ContentID   uint `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	MaxAttempts int  `json:"maxAttempts" gorm:"not null;default:1"`
	PassPercent int  `json:"passPercent" gorm:"not null;default:50"`
	Published   bool `json:"published" gorm:"not null;default:false"`
	// NotifiedAt is set when students were told the quiz is open. Publish clears it so a republish notifies again.
	NotifiedAt *time.Time `json:"-"`
	// NotifyUpdate selects the wording of the next announcement: true means "quiz updated", false "new quiz".
	NotifyUpdate bool       `json:"-" gorm:"not null;default:false"`
	Questions    []Question `json:"questions,omitempty" gorm:"foreignKey:QuizID;references:ContentID;constraint:OnDelete:CASCADE;"`
}

// Question represents an individual quiz prompt supporting hints and automated grading criteria.
type Question struct {
	gorm.Model
	Uuid      uuid.UUID    `json:"uuid" gorm:"type:uuid;index"`
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

func (q *Question) BeforeCreate(tx *gorm.DB) error {
	if q.Uuid == uuid.Nil {
		q.Uuid = uuid.New()
	}
	return nil
}

// CodingExercise is an Assignment subtype: an automated coding task with starter code repo and test suites.
type CodingExercise struct {
	ContentID       uint   `json:"contentId" gorm:"primaryKey;autoIncrement:false"`
	StarterCodeRepo string `json:"starterCodeRepo" gorm:"type:varchar(255);not null"`
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
