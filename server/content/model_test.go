package content_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContentAggregate_ClassTableInheritanceAndPreloads(t *testing.T) {
	db := testutil.NewSqliteTestDb(
		t,
		&user.User{},
		&course.Course{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.Assignment{},
		&content.Quiz{},
		&content.Question{},
		&content.CodingExercise{},
		&content.StudentInteraction{},
	)
	assert.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	var assignmentsDDL, quizzesDDL, codingExercisesDDL string
	assert.NoError(t, db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='assignments'").Scan(&assignmentsDDL).Error)
	assert.NoError(t, db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='quizzes'").Scan(&quizzesDDL).Error)
	assert.NoError(t, db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='coding_exercises'").Scan(&codingExercisesDDL).Error)

	assert.Contains(t, assignmentsDDL, "REFERENCES `contents`(`id`)")
	assert.NotContains(t, assignmentsDDL, "fk_quizzes_assignment")
	assert.NotContains(t, assignmentsDDL, "fk_coding_exercises_assignment")
	assert.Contains(t, quizzesDDL, "REFERENCES `assignments`(`content_id`)")
	assert.Contains(t, codingExercisesDDL, "REFERENCES `assignments`(`content_id`)")

	u := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_alice",
		Email:        "alice@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(u).Error)

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	// 1. File with iframe source URL
	cntPage := &content.Content{
		ModuleID: m.ID,
		Title:    "Syllabus (Embedded)",
		Position: 1,
		Type:     content.TypeFile,
	}
	assert.NoError(t, db.Create(cntPage).Error)
	assert.NotEqual(t, uuid.Nil, cntPage.Uuid, "uuids don't match")

	page := &content.File{
		ContentID: cntPage.ID,
		SourceURL: "https://docs.example.com/syllabus-embed.html",
		RichText:  "Embedded course syllabus iframe",
	}
	assert.NoError(t, db.Create(page).Error)

	// 2. Quiz as Assignment subtype
	cntQuiz := &content.Content{
		ModuleID: m.ID,
		Title:    "Chapter 1 Quiz",
		Position: 2,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cntQuiz).Error)
	assert.NotEqual(t, uuid.Nil, cntQuiz.Uuid)

	deadline := time.Now().Add(24 * time.Hour)
	asgnQuiz := &content.Assignment{
		ContentID: cntQuiz.ID,
		Type:      content.AssignmentQuiz,
		Deadline:  &deadline,
	}
	assert.NoError(t, db.Create(asgnQuiz).Error)

	qz := &content.Quiz{
		ContentID:   asgnQuiz.ContentID,
		MaxAttempts: 2,
		Questions: []content.Question{
			{
				Position:  1,
				Type:      content.TypeMultipleChoice,
				Prompt:    "What is GORM?",
				Options:   `["ORM","Queue"]`,
				AnswerKey: "ORM",
				Points:    5.0,
			},
		},
	}
	assert.NoError(t, db.Create(qz).Error)

	// 3. CodingExercise as Assignment subtype
	cntEx := &content.Content{
		ModuleID: m.ID,
		Title:    "Lab 1",
		Position: 3,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cntEx).Error)

	asgnEx := &content.Assignment{
		ContentID:              cntEx.ID,
		Type:                   content.AssignmentCodingExercise,
		AllowsGroupSubmission:  true,
		AcceptsLateSubmissions: false,
	}
	assert.NoError(t, db.Create(asgnEx).Error)

	ex := &content.CodingExercise{
		ContentID:        asgnEx.ContentID,
		StarterCodeRepo:  "https://git.app.uib.no/lab1.git",
		VisibleTestCases: "TestMain",
	}
	assert.NoError(t, db.Create(ex).Error)

	// 4. Pure HandIn Assignment (no subtype extension table)
	cntHandIn := &content.Content{
		ModuleID: m.ID,
		Title:    "Essay 1",
		Position: 4,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cntHandIn).Error)

	asgnHandIn := &content.Assignment{
		ContentID:              cntHandIn.ID,
		Type:                   content.AssignmentHandIn,
		Deadline:               &deadline,
		AllowsGroupSubmission:  false,
		AcceptsLateSubmissions: true,
	}
	assert.NoError(t, db.Create(asgnHandIn).Error)

	// 5. StudentInteraction
	now := time.Now()
	inter := &content.StudentInteraction{
		UserID:      u.ID,
		UserEmail:   u.Email,
		ContentID:   cntPage.ID,
		CompletedAt: &now,
	}
	assert.NoError(t, db.Create(inter).Error)

	// Preload directly from Content
	var foundPage content.Content
	err := db.Preload("File").Preload("Module.Course").Preload("Interactions").First(&foundPage, cntPage.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundPage.File)
	assert.Equal(t, "https://docs.example.com/syllabus-embed.html", foundPage.File.SourceURL)
	assert.Len(t, foundPage.Interactions, 1)

	// Preload Assignment -> Quiz -> Questions
	var foundQuiz content.Content
	err = db.Preload("Assignment.Quiz.Questions").First(&foundQuiz, cntQuiz.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundQuiz.Assignment)
	assert.Equal(t, content.AssignmentQuiz, foundQuiz.Assignment.Type)
	assert.NotNil(t, foundQuiz.Assignment.Quiz)
	assert.Len(t, foundQuiz.Assignment.Quiz.Questions, 1)

	// Preload Assignment -> CodingExercise
	var foundEx content.Content
	err = db.Preload("Assignment.CodingExercise").First(&foundEx, cntEx.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundEx.Assignment)
	assert.Equal(t, content.AssignmentCodingExercise, foundEx.Assignment.Type)
	assert.NotNil(t, foundEx.Assignment.CodingExercise)
	assert.Equal(t, "https://git.app.uib.no/lab1.git", foundEx.Assignment.CodingExercise.StarterCodeRepo)

	// Preload Assignment HandIn
	var foundHandIn content.Content
	err = db.Preload("Assignment").First(&foundHandIn, cntHandIn.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundHandIn.Assignment)
	assert.Equal(t, content.AssignmentHandIn, foundHandIn.Assignment.Type)
	assert.Nil(t, foundHandIn.Assignment.Quiz)
	assert.Nil(t, foundHandIn.Assignment.CodingExercise)
}
