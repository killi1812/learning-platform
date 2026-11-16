package grade

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"github.com/google/uuid"
)

// UpsertGradeDto defines the payload sent by a teacher to grade or update a submission.
type UpsertGradeDto struct {
	Score              *float64 `json:"score" binding:"required,gte=0" example:"85.5"`
	Feedback           string   `json:"feedback" binding:"max=10000" example:"Good overall structure, minor edge-case bug."`
	IsVisibleToStudent bool     `json:"isVisibleToStudent" example:"true"`
}

// GradeDto represents the full grade details for a submission.
type GradeDto struct {
	Score              float64    `json:"score"`
	MaxPoints          float64    `json:"maxPoints"`
	Feedback           string     `json:"feedback,omitempty"`
	IsVisibleToStudent bool       `json:"isVisibleToStudent"`
	GradedByEmail      string     `json:"gradedByEmail,omitempty"`
	GradedAt           time.Time  `json:"gradedAt"`
	ReleasedAt         *time.Time `json:"releasedAt,omitempty"`
}

// ReleaseResultDto indicates how many grades were transitioned to visible.
type ReleaseResultDto struct {
	Released int `json:"released"`
}

// PendingSubmissionDto represents an ungraded submission row for the teacher dashboard.
type PendingSubmissionDto struct {
	SubmissionUuid  uuid.UUID  `json:"submissionUuid"`
	CourseUuid      uuid.UUID  `json:"courseUuid"`
	ContentUuid     uuid.UUID  `json:"contentUuid"`
	CourseCode      string     `json:"courseCode"`
	CourseName      string     `json:"courseName"`
	AssignmentTitle string     `json:"assignmentTitle"`
	UserEmail       string     `json:"userEmail"`
	UserName        string     `json:"userName,omitempty"`
	Attempt         int        `json:"attempt"`
	IsLate          bool       `json:"isLate"`
	SubmittedAt     time.Time  `json:"submittedAt"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	MaxPoints       float64    `json:"maxPoints"`
}

// StudentGradeType distinguishes manual hand-in grades from quiz auto-grades.
type StudentGradeType string

const (
	GradeTypeHandin StudentGradeType = "HANDIN"
	GradeTypeQuiz   StudentGradeType = "QUIZ"
)

// StudentGradeDto represents a single graded item (hand-in or quiz) in the student's grade overview.
type StudentGradeDto struct {
	Type            StudentGradeType `json:"type"`
	CourseUuid      uuid.UUID        `json:"courseUuid"`
	ContentUuid     uuid.UUID        `json:"contentUuid"`
	CourseName      string           `json:"courseName"`
	CourseCode      string           `json:"courseCode"`
	ModuleTitle     string           `json:"moduleTitle"`
	AssignmentTitle string           `json:"assignmentTitle"`
	Attempt         int              `json:"attempt"`
	Score           float64          `json:"score"`
	MaxPoints       float64          `json:"maxPoints"`
	Percent         float64          `json:"percent"`
	Feedback        string           `json:"feedback,omitempty"`
	GradedAt        time.Time        `json:"gradedAt"`
}

// CourseFinalGradeDto represents a student's cumulative evaluation for an entire subject/course.
type CourseFinalGradeDto struct {
	CourseUuid    uuid.UUID              `json:"courseUuid"`
	CourseName    string                 `json:"courseName"`
	CourseCode    string                 `json:"courseCode"`
	TotalScore    float64                `json:"totalScore"`
	TotalMaxScore float64                `json:"totalMaxScore"`
	Percent       float64                `json:"percent"`
	FinalGrade    string                 `json:"finalGrade"` // "A", "B", "C", "D", "E", "F"
	GradingScale  course.GradingScaleDto `json:"gradingScale"`
}

// StudentFinalGradeRowDto represents a student's final grade in the teacher's course roster overview.
type StudentFinalGradeRowDto struct {
	UserUuid      uuid.UUID `json:"userUuid"`
	UserEmail     string    `json:"userEmail"`
	UserName      string    `json:"userName,omitempty"`
	TotalScore    float64   `json:"totalScore"`
	TotalMaxScore float64   `json:"totalMaxScore"`
	Percent       float64   `json:"percent"`
	FinalGrade    string    `json:"finalGrade"`
}
