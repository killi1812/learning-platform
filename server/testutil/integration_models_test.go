package testutil_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAllLmsModels_EndToEndAutoMigrateAndWorkflow(t *testing.T) {
	// 1. AutoMigrate all 3 domain aggregates together
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		// Course aggregate
		&course.Course{},
		&course.Enrollment{},
		&course.Group{},
		&course.Module{},
		// Content aggregate
		&content.Content{},
		&content.StaticPage{},
		&content.Assignment{},
		&content.Quiz{},
		&content.Question{},
		&content.CodingExercise{},
		&content.StudentInteraction{},
		// Submission aggregate
		&submission.Submission{},
		&submission.GroupSubmission{},
		&submission.IndividualSubmission{},
		&submission.Grade{},
	)

	// 2. Create Users (Teacher and Student)
	teacher := &user.User{
		Uuid:         uuid.New(),
		Username:     "prof_smith",
		Email:        "prof@hvl.no",
		PasswordHash: "secret1",
		Role:         authutils.ROLE_USER,
	}
	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "alice",
		Email:        "alice@hvl.no",
		PasswordHash: "secret2",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(teacher).Error)
	assert.NoError(t, db.Create(student).Error)

	// 3. Create Course
	c := &course.Course{
		Title:       "DAT250: Distributed Systems",
		Description: "Advanced enterprise software course",
		Term:        course.TermAutumn,
		Year:        2026,
	}
	assert.NoError(t, db.Create(c).Error)

	// 4. Enrollments (Per-course roles: prof is teacher, alice is student)
	enrTeacher := &course.Enrollment{
		UserID:     teacher.ID,
		UserEmail:  teacher.Email,
		CourseID:   c.ID,
		Role:       course.RoleTeacher,
		EnrolledAt: time.Now(),
	}
	enrStudent := &course.Enrollment{
		UserID:     student.ID,
		UserEmail:  student.Email,
		CourseID:   c.ID,
		Role:       course.RoleStudent,
		EnrolledAt: time.Now(),
	}
	assert.NoError(t, db.Create(enrTeacher).Error)
	assert.NoError(t, db.Create(enrStudent).Error)

	// 5. Group formation
	grp := &course.Group{
		CourseID: c.ID,
		Name:     "Distributed Group 1",
		Members:  []user.User{*student},
	}
	assert.NoError(t, db.Create(grp).Error)

	// 6. Ordered Modules with dependency
	m1 := &course.Module{
		CourseID: c.ID,
		Title:    "Module 1: Foundations",
		Position: 1,
	}
	assert.NoError(t, db.Create(m1).Error)

	m2 := &course.Module{
		CourseID:    c.ID,
		Title:       "Module 2: Advanced Topics",
		Position:    2,
		DependsOnID: &m1.ID,
	}
	assert.NoError(t, db.Create(m2).Error)

	// 7. Base Content items with specialized tables
	// 7a. Static Page using iframe SourceURL
	cntPage := &content.Content{
		ModuleID: m1.ID,
		Title:    "Syllabus & Lecture Notes",
		Position: 1,
		Type:     content.TypeStaticPage,
	}
	assert.NoError(t, db.Create(cntPage).Error)
	page := &content.StaticPage{
		ContentID: cntPage.ID,
		SourceURL: "https://docs.example.com/syllabus.html",
		RichText:  "Embedded notes",
	}
	assert.NoError(t, db.Create(page).Error)

	// 7b. Quiz
	cntQuiz := &content.Content{
		ModuleID: m1.ID,
		Title:    "Readiness Quiz",
		Position: 2,
		Type:     content.TypeAssignment,
	}
	assert.NoError(t, db.Create(cntQuiz).Error)
	asgnQuiz := &content.Assignment{
		ContentID: cntQuiz.ID,
		Type:      content.AssignmentQuiz,
	}
	assert.NoError(t, db.Create(asgnQuiz).Error)
	qz := &content.Quiz{
		ContentID:   asgnQuiz.ContentID,
		MaxAttempts: 2,
		Questions: []content.Question{
			{
				Position:  1,
				Type:      content.TypeMultipleChoice,
				Prompt:    "What does ORM stand for?",
				Options:   `["Object-Relational Mapping","Online Record Management"]`,
				AnswerKey: "Object-Relational Mapping",
				Points:    5.0,
			},
		},
	}
	assert.NoError(t, db.Create(qz).Error)

	// 7c. Coding Exercise
	cntEx := &content.Content{
		ModuleID: m2.ID,
		Title:    "Exercise 1: Raft Implementation",
		Position: 1,
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
		StarterCodeRepo:  "https://git.app.uib.no/exercises/raft.git",
		VisibleTestCases: "TestLeaderElection",
		HiddenTestCases:  "TestNetworkPartition",
	}
	assert.NoError(t, db.Create(ex).Error)

	// 8. Student Interaction (unlocking criteria)
	now := time.Now()
	inter := &content.StudentInteraction{
		UserID:      student.ID,
		UserEmail:   student.Email,
		ContentID:   cntPage.ID,
		CompletedAt: &now,
	}
	assert.NoError(t, db.Create(inter).Error)

	// 9. Submissions (Group submission for coding exercise, Individual submission for quiz)
	grpSub := &submission.GroupSubmission{
		Submission: submission.Submission{
			ContentID:   cntEx.ID,
			Attempt:     1,
			IsLate:      false,
			PayloadURL:  "https://git.app.uib.no/group1/raft-submission.git",
			SubmittedAt: time.Now(),
		},
		GroupID: &grp.ID,
		UserID:  &student.ID,
	}
	assert.NoError(t, db.Create(grpSub).Error)

	indivSub := &submission.IndividualSubmission{
		Submission: submission.Submission{
			ContentID:   cntQuiz.ID,
			Attempt:     1,
			IsLate:      false,
			PayloadURL:  "quiz-attempt-1",
			SubmittedAt: time.Now(),
		},
		UserID:    &student.ID,
		UserEmail: student.Email,
	}
	assert.NoError(t, db.Create(indivSub).Error)

	sub := &submission.Submission{
		ContentID:   cntEx.ID,
		Attempt:     1,
		IsLate:      false,
		PayloadURL:  "https://git.app.uib.no/student/raft-submission.git",
		SubmittedAt: time.Now(),
	}
	assert.NoError(t, db.Create(sub).Error)

	// 10. Grade & Feedback
	grd := &submission.Grade{
		SubmissionID:       sub.ID,
		Score:              100.0,
		Feedback:           "Flawless election logic under partitions.",
		IsVisibleToStudent: true,
		GradedByID:         &teacher.ID,
	}
	assert.NoError(t, db.Create(grd).Error)

	// 11. Assert complete relational integrity and preloads across aggregates
	var loadedCourse course.Course
	err := db.
		Preload("Enrollments.User").
		Preload("Groups.Members").
		Preload("Modules").
		First(&loadedCourse, c.ID).Error
	assert.NoError(t, err)
	assert.Len(t, loadedCourse.Modules, 2)
	assert.Len(t, loadedCourse.Groups, 1)

	var loadedContent content.Content
	err = db.
		Preload("StaticPage").
		Preload("Module.Course").
		Preload("Interactions.User").
		First(&loadedContent, cntPage.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, loadedContent.StaticPage)
	assert.Equal(t, "https://docs.example.com/syllabus.html", loadedContent.StaticPage.SourceURL)
	assert.Equal(t, "DAT250: Distributed Systems", loadedContent.Module.Course.Title)

	var loadedGroupSub submission.GroupSubmission
	err = db.
		Preload("Group.Members").
		Preload("User").
		Preload("Content.Module.Course").
		First(&loadedGroupSub, grpSub.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, loadedGroupSub.Group)
	assert.Equal(t, "Distributed Group 1", loadedGroupSub.Group.Name)
	assert.Len(t, loadedGroupSub.Group.Members, 1)
	assert.NotNil(t, loadedGroupSub.User)
	assert.Equal(t, "alice@hvl.no", loadedGroupSub.User.Email)
	assert.NotNil(t, loadedGroupSub.Content)
	assert.Equal(t, "Exercise 1: Raft Implementation", loadedGroupSub.Content.Title)

	var loadedIndivSub submission.IndividualSubmission
	err = db.
		Preload("User").
		Preload("Content").
		First(&loadedIndivSub, indivSub.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, loadedIndivSub.User)
	assert.Equal(t, "alice@hvl.no", loadedIndivSub.User.Email)
	assert.Equal(t, "Readiness Quiz", loadedIndivSub.Content.Title)

	var loadedGrade submission.Grade
	err = db.
		Preload("Submission.Content.Module.Course").
		Preload("GradedBy").
		First(&loadedGrade, grd.ID).Error

	assert.NoError(t, err)
	assert.Equal(t, 100.0, loadedGrade.Score)
	assert.True(t, loadedGrade.IsVisibleToStudent)
	assert.Equal(t, "prof@hvl.no", loadedGrade.GradedBy.Email)
	assert.NotNil(t, loadedGrade.Submission)
	assert.Equal(t, "Exercise 1: Raft Implementation", loadedGrade.Submission.Content.Title)
}
