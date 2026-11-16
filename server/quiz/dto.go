package quiz

import (
	"encoding/json"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
)

// --- Teacher requests ---

// CreateQuizRequest creates a draft quiz inside a module.
type CreateQuizRequest struct {
	Title string `json:"title" binding:"required,min=2,max=255"`
}

// UpdateQuizRequest changes quiz settings. Dates are the availability window.
type UpdateQuizRequest struct {
	Title        string     `json:"title" binding:"required,min=2,max=255"`
	VisibleFrom  *time.Time `json:"visibleFrom"`
	VisibleUntil *time.Time `json:"visibleUntil"`
	PassPercent  int        `json:"passPercent" binding:"min=0,max=100"`
	MaxAttempts  int        `json:"maxAttempts" binding:"min=1,max=100"`
}

// QuestionInput is one question as the teacher edits it. Array order is question order.
// Correct holds the option indices that are marked correct.
type QuestionInput struct {
	Type    content.QuestionType `json:"type" binding:"required"`
	Prompt  string               `json:"prompt" binding:"required,min=1,max=1000"`
	Options []string             `json:"options"`
	Correct []int                `json:"correct"`
}

// ReplaceQuestionsRequest replaces the whole ordered question list.
type ReplaceQuestionsRequest struct {
	Questions []QuestionInput `json:"questions" binding:"required,max=100,dive"`
}

// --- Teacher responses (contain the answer key, teacher endpoints only) ---

type EditQuestionDto struct {
	Uuid    string               `json:"uuid"`
	Type    content.QuestionType `json:"type"`
	Prompt  string               `json:"prompt"`
	Options []string             `json:"options"`
	Correct []int                `json:"correct"`
}

type EditQuizDto struct {
	Uuid         string            `json:"uuid"`
	ModuleUuid   string            `json:"moduleUuid"`
	Title        string            `json:"title"`
	Published    bool              `json:"published"`
	VisibleFrom  *time.Time        `json:"visibleFrom,omitempty"`
	VisibleUntil *time.Time        `json:"visibleUntil,omitempty"`
	PassPercent  int               `json:"passPercent"`
	MaxAttempts  int               `json:"maxAttempts"`
	Questions    []EditQuestionDto `json:"questions"`
}

type StudentResultRowDto struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Attempts int      `json:"attempts"`
	Percent  *float64 `json:"percent,omitempty"`
	Passed   *bool    `json:"passed,omitempty"`
}

// --- Student responses (never contain the answer key) ---

type StudentQuestionDto struct {
	Uuid    string               `json:"uuid"`
	Type    content.QuestionType `json:"type"`
	Prompt  string               `json:"prompt"`
	Options []string             `json:"options"`
}

type AttemptDto struct {
	Questions []StudentQuestionDto `json:"questions"`
	Answers   map[string][]int     `json:"answers"` // question uuid -> selected option indices
	StartedAt time.Time            `json:"startedAt"`
}

type SaveAnswerRequest struct {
	Selected []int `json:"selected" binding:"required"`
}

// ResultDto is the only thing a student learns about a graded quiz.
type ResultDto struct {
	Percent      float64 `json:"percent"`
	Passed       bool    `json:"passed"`
	AttemptsUsed int     `json:"attemptsUsed"`
	MaxAttempts  int     `json:"maxAttempts"`
}

// AssignmentDto is one row in the course assignment list.
type AssignmentDto struct {
	Uuid                   string     `json:"uuid"`
	Type                   string     `json:"type"`
	Title                  string     `json:"title"`
	ModuleUuid             string     `json:"moduleUuid"`
	ModuleTitle            string     `json:"moduleTitle"`
	Published              bool       `json:"published"`
	VisibleFrom            *time.Time `json:"visibleFrom,omitempty"`
	VisibleUntil           *time.Time `json:"visibleUntil,omitempty"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission"`
	MaxPoints              float64    `json:"maxPoints,omitempty"`
	PassPercent            int        `json:"passPercent"`
	MaxAttempts            int        `json:"maxAttempts"`
	MyResult               *ResultDto `json:"myResult,omitempty"`
	AttemptsUsed           int        `json:"attemptsUsed"`
}

// decodeOptions parses the JSON option list stored on a question.
func decodeOptions(raw string) []string {
	options := []string{}
	if raw == "" {
		return options
	}
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return []string{}
	}
	return options
}

// decodeIndices parses a JSON index list (answer key or student selection).
func decodeIndices(raw string) []int {
	indices := []int{}
	if raw == "" {
		return indices
	}
	if err := json.Unmarshal([]byte(raw), &indices); err != nil {
		return []int{}
	}
	return indices
}

func toEditQuestion(q content.Question) EditQuestionDto {
	return EditQuestionDto{
		Uuid:    q.Uuid.String(),
		Type:    q.Type,
		Prompt:  q.Prompt,
		Options: decodeOptions(q.Options),
		Correct: decodeIndices(q.AnswerKey),
	}
}

// toStudentQuestion deliberately drops AnswerKey.
func toStudentQuestion(q content.Question) StudentQuestionDto {
	return StudentQuestionDto{
		Uuid:    q.Uuid.String(),
		Type:    q.Type,
		Prompt:  q.Prompt,
		Options: decodeOptions(q.Options),
	}
}
