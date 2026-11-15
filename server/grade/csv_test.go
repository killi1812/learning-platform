package grade

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"normal text", "Hello World", "Hello World"},
		{"formula equals", "=SUM(A1:A10)", "'=SUM(A1:A10)"},
		{"formula plus", "+12345", "'+12345"},
		{"formula minus", "-500", "'-500"},
		{"formula at", "@cmd|' /C calc'!A0", "'@cmd|' /C calc'!A0"},
		{"tab prefix", "\tmalicious", "'\tmalicious"},
		{"carriage return prefix", "\rmalicious", "'\rmalicious"},
		{"newline prefix", "\n=cmd", "'\n=cmd"},
		{"leading space before equals", " =SUM(A1)", "' =SUM(A1)"},
		{"leading spaces before plus", "  +1+1", "'  +1+1"},
		{"leading spaces normal text untouched", "  Hello", "  Hello"},
		{"normal email", "student@example.com", "student@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeCSVCell(tt.input))
		})
	}
}

func TestWriteGradebookCSV(t *testing.T) {
	t.Run("writes header and sanitized records", func(t *testing.T) {
		var buf bytes.Buffer
		records := []GradebookRecord{
			{
				StudentEmail:       "student@example.com",
				StudentName:        "Alice",
				ModuleTitle:        "Module 1",
				AssignmentTitle:    "Assignment 1",
				AssignmentType:     "HANDIN",
				Attempt:            "1",
				SubmittedAt:        "2026-11-14T10:00:00Z",
				IsLate:             "false",
				Score:              "95.00",
				MaxPoints:          "100.00",
				Percent:            "95.00%",
				IsVisibleToStudent: "true",
				GradedByEmail:      "teacher@example.com",
				GradedAt:           "2026-11-14T12:00:00Z",
				Feedback:           "Line 1\nLine 2",
				FinalCourseScore:   "95.00",
				FinalCourseMax:     "100.00",
				FinalCoursePercent: "95.00%",
				FinalCourseGrade:   "A",
			},
			{
				StudentEmail:       "malicious@example.com",
				StudentName:        "=cmd",
				ModuleTitle:        "Module 2",
				AssignmentTitle:    "Quiz 1",
				AssignmentType:     "QUIZ",
				Attempt:            "1",
				Score:              "80.00",
				MaxPoints:          "100.00",
				Percent:            "80.00%",
				IsVisibleToStudent: "true",
				Feedback:           "@danger",
				FinalCourseScore:   "80.00",
				FinalCourseMax:     "100.00",
				FinalCoursePercent: "80.00%",
				FinalCourseGrade:   "B",
			},
		}

		err := WriteGradebookCSV(&buf, records)
		assert.NoError(t, err)

		out := buf.String()
		assert.Contains(t, out, "student_email,student_name,module,assignment,type,attempt,submitted_at,is_late,score,max_points,percent,visible_to_student,graded_by,graded_at,feedback,final_course_score,final_course_max,final_course_percent,final_course_grade")
		assert.Contains(t, out, "student@example.com,Alice,Module 1,Assignment 1,HANDIN,1,2026-11-14T10:00:00Z,false,95.00,100.00,95.00%,true,teacher@example.com,2026-11-14T12:00:00Z,Line 1 Line 2,95.00,100.00,95.00%,A")
		// Formula sanitization check
		assert.Contains(t, out, "'=cmd")
		assert.Contains(t, out, "'@danger")
	})

	t.Run("normalizes newlines and leading whitespace formulas in all text fields", func(t *testing.T) {
		var buf bytes.Buffer
		records := []GradebookRecord{
			{
				StudentEmail:    "alice@example.com",
				StudentName:     "Alice\r\nSmith",
				ModuleTitle:     "\r\n=IMPORTXML()",
				AssignmentTitle: "HW\r1",
				Feedback:        "Line 1\r\nLine 2",
			},
		}

		err := WriteGradebookCSV(&buf, records)
		assert.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, lines, 2)
		assert.Contains(t, lines[1], "Alice Smith")
		assert.Contains(t, lines[1], "' =IMPORTXML()")
		assert.Contains(t, lines[1], "HW 1")
		assert.Contains(t, lines[1], "Line 1 Line 2")
	})

	t.Run("empty records produces only header", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteGradebookCSV(&buf, nil)
		assert.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, lines, 1)
		assert.Contains(t, lines[0], "student_email")
	})
}
