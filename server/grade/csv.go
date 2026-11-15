package grade

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// SanitizeCSVCell guards against CSV formula injection (OWASP).
// If a cell value starts with '=', '+', '-', '@', '\t', or '\r',
// it prepends a single quote so spreadsheet programs treat it as raw text.
func SanitizeCSVCell(value string) string {
	if len(value) == 0 {
		return ""
	}
	first := value[0]
	if first == '=' || first == '+' || first == '-' || first == '@' || first == '\t' || first == '\r' {
		return "'" + value
	}
	return value
}

// GradebookRecord holds the resolved data for one student row in a course grade export.
type GradebookRecord struct {
	StudentEmail       string
	StudentName        string
	ModuleTitle        string
	AssignmentTitle    string
	AssignmentType     string
	Attempt            string
	SubmittedAt        string
	IsLate             string
	Score              string
	MaxPoints          string
	Percent            string
	IsVisibleToStudent string
	GradedByEmail      string
	GradedAt           string
	Feedback           string
	FinalCourseScore   string
	FinalCourseMax     string
	FinalCoursePercent string
	FinalCourseGrade   string
}

// WriteGradebookCSV writes a slice of gradebook records as a CSV into w.
func WriteGradebookCSV(w io.Writer, records []GradebookRecord) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"student_email",
		"student_name",
		"module",
		"assignment",
		"type",
		"attempt",
		"submitted_at",
		"is_late",
		"score",
		"max_points",
		"percent",
		"visible_to_student",
		"graded_by",
		"graded_at",
		"feedback",
		"final_course_score",
		"final_course_max",
		"final_course_percent",
		"final_course_grade",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, rec := range records {
		row := []string{
			SanitizeCSVCell(rec.StudentEmail),
			SanitizeCSVCell(rec.StudentName),
			SanitizeCSVCell(rec.ModuleTitle),
			SanitizeCSVCell(rec.AssignmentTitle),
			SanitizeCSVCell(rec.AssignmentType),
			SanitizeCSVCell(rec.Attempt),
			SanitizeCSVCell(rec.SubmittedAt),
			SanitizeCSVCell(rec.IsLate),
			SanitizeCSVCell(rec.Score),
			SanitizeCSVCell(rec.MaxPoints),
			SanitizeCSVCell(rec.Percent),
			SanitizeCSVCell(rec.IsVisibleToStudent),
			SanitizeCSVCell(rec.GradedByEmail),
			SanitizeCSVCell(rec.GradedAt),
			SanitizeCSVCell(strings.ReplaceAll(rec.Feedback, "\n", " ")),
			SanitizeCSVCell(rec.FinalCourseScore),
			SanitizeCSVCell(rec.FinalCourseMax),
			SanitizeCSVCell(rec.FinalCoursePercent),
			SanitizeCSVCell(rec.FinalCourseGrade),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}
