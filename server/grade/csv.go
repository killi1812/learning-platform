package grade

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// SanitizeCSVCell guards against CSV formula injection (OWASP).
// If a cell value starts with '=', '+', '-', '@', '\t', '\r', or '\n' (or leading
// whitespace before '=', '+', '-', '@'), it prepends a single quote so spreadsheet
// programs treat it as raw text.
func SanitizeCSVCell(value string) string {
	if len(value) == 0 {
		return ""
	}
	first := value[0]
	if isFormulaPrefix(first) || first == '\t' || first == '\r' || first == '\n' {
		return "'" + value
	}
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if len(trimmed) > 0 && isFormulaPrefix(trimmed[0]) {
		return "'" + value
	}
	return value
}

func isFormulaPrefix(b byte) bool {
	return b == '=' || b == '+' || b == '-' || b == '@'
}

func sanitizeCSVTextCell(value string) string {
	cleaned := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(value)
	return SanitizeCSVCell(cleaned)
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
			sanitizeCSVTextCell(rec.StudentEmail),
			sanitizeCSVTextCell(rec.StudentName),
			sanitizeCSVTextCell(rec.ModuleTitle),
			sanitizeCSVTextCell(rec.AssignmentTitle),
			sanitizeCSVTextCell(rec.AssignmentType),
			sanitizeCSVTextCell(rec.Attempt),
			sanitizeCSVTextCell(rec.SubmittedAt),
			sanitizeCSVTextCell(rec.IsLate),
			sanitizeCSVTextCell(rec.Score),
			sanitizeCSVTextCell(rec.MaxPoints),
			sanitizeCSVTextCell(rec.Percent),
			sanitizeCSVTextCell(rec.IsVisibleToStudent),
			sanitizeCSVTextCell(rec.GradedByEmail),
			sanitizeCSVTextCell(rec.GradedAt),
			sanitizeCSVTextCell(rec.Feedback),
			sanitizeCSVTextCell(rec.FinalCourseScore),
			sanitizeCSVTextCell(rec.FinalCourseMax),
			sanitizeCSVTextCell(rec.FinalCoursePercent),
			sanitizeCSVTextCell(rec.FinalCourseGrade),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}
