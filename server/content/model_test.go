package content_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContentAggregate_ClassTableInheritanceAndPreloads(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Module{},
		&content.Content{},
		&content.StaticPage{},
		&content.Quiz{},
		&content.Question{},
		&content.Assignment{},
		&content.CodingExercise{},
		&content.StudentInteraction{},
	)

	u := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_alice",
		Email:        "alice@hvl.no",
		PasswordHash: "pass",
		Role:         auth.ROLE_USER,
	}
	assert.NoError(t, db.Create(u).Error)

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	// 1. StaticPage with iframe source URL
	cntPage := &content.Content{
		ModuleID: m.ID,
		Title:    "Syllabus (Embedded)",
		Position: 1,
		Type:     content.TypeStaticPage,
	}
	assert.NoError(t, db.Create(cntPage).Error)
	page := &content.StaticPage{
		ContentID: cntPage.ID,
		SourceURL: "https://docs.example.com/syllabus-embed.html",
		RichText:  "Embedded course syllabus iframe",
	}
	assert.NoError(t, db.Create(page).Error)

	// 2. Quiz with Questions
	cntQuiz := &content.Content{
		ModuleID: m.ID,
		Title:    "Chapter 1 Quiz",
		Position: 2,
		Type:     content.TypeQuiz,
	}
	assert.NoError(t, db.Create(cntQuiz).Error)
	deadline := time.Now().Add(24 * time.Hour)
	qz := &content.Quiz{
		ContentID:   cntQuiz.ID,
		MaxAttempts: 2,
		Deadline:    &deadline,
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

	// 3. CodingExercise
	cntEx := &content.Content{
		ModuleID: m.ID,
		Title:    "Lab 1",
		Position: 3,
		Type:     content.TypeCodingExercise,
	}
	assert.NoError(t, db.Create(cntEx).Error)
	ex := &content.CodingExercise{
		ContentID:              cntEx.ID,
		StarterCodeRepo:        "https://git.app.uib.no/lab1.git",
		AllowsGroupSubmission:  true,
		AcceptsLateSubmissions: false,
		VisibleTestCases:       "TestMain",
	}
	assert.NoError(t, db.Create(ex).Error)

	// 4. StudentInteraction
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
	err := db.Preload("StaticPage").Preload("Module.Course").Preload("Interactions").First(&foundPage, cntPage.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundPage.StaticPage)
	assert.Equal(t, "https://docs.example.com/syllabus-embed.html", foundPage.StaticPage.SourceURL)
	assert.Len(t, foundPage.Interactions, 1)

	var foundQuiz content.Content
	err = db.Preload("Quiz.Questions").First(&foundQuiz, cntQuiz.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundQuiz.Quiz)
	assert.Len(t, foundQuiz.Quiz.Questions, 1)
}
