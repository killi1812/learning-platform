package submission_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSubmissionAggregate_IndividualGroupAndGrading(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Group{},
		&course.Module{},
		&content.Content{},
		&submission.Submission{},
		&submission.GroupSubmission{},
		&submission.IndividualSubmission{},
		&submission.Grade{},
	)

	teacher := &user.User{
		Uuid:         uuid.New(),
		Username:     "teacher_t",
		Email:        "teacher@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_s",
		Email:        "student@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(teacher).Error)
	assert.NoError(t, db.Create(student).Error)

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	grp := &course.Group{CourseID: c.ID, Name: "Team Alpha"}
	assert.NoError(t, db.Create(grp).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	cnt := &content.Content{
		ModuleID: m.ID,
		Title:    "Exercise 1",
		Position: 1,
		Type:     content.TypeCodingExercise,
	}
	assert.NoError(t, db.Create(cnt).Error)

	// Group Submission
	grpSub := &submission.GroupSubmission{
		Submission: submission.Submission{
			ContentID:  cnt.ID,
			Attempt:    1,
			IsLate:     false,
			PayloadURL: "https://git.app.uib.no/team-alpha/exercise1.git",
		},
		GroupID: &grp.ID,
		UserID:  &student.ID,
	}
	assert.NoError(t, db.Create(grpSub).Error)

	// Individual Submission
	indivSub := &submission.IndividualSubmission{
		Submission: submission.Submission{
			ContentID:  cnt.ID,
			Attempt:    1,
			IsLate:     false,
			PayloadURL: "https://git.app.uib.no/student/exercise1.git",
		},
		UserID:    &student.ID,
		UserEmail: student.Email,
	}
	assert.NoError(t, db.Create(indivSub).Error)

	// Preload Group Submission
	var foundGrpSub submission.GroupSubmission
	err := db.Preload("Group").Preload("User").Preload("Content").First(&foundGrpSub, grpSub.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundGrpSub.Group)
	assert.Equal(t, "Team Alpha", foundGrpSub.Group.Name)
	assert.NotNil(t, foundGrpSub.User)
	assert.Equal(t, "student@hvl.no", foundGrpSub.User.Email)
	assert.NotNil(t, foundGrpSub.Content)
	assert.Equal(t, "Exercise 1", foundGrpSub.Content.Title)

	// Preload Individual Submission
	var foundIndivSub submission.IndividualSubmission
	err = db.Preload("User").Preload("Content").First(&foundIndivSub, indivSub.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundIndivSub.User)
	assert.Equal(t, "student@hvl.no", foundIndivSub.User.Email)
	assert.Equal(t, student.Email, foundIndivSub.UserEmail)

	// Base Submission & Grade
	sub := &submission.Submission{
		ContentID:  cnt.ID,
		Attempt:    1,
		IsLate:     false,
		PayloadURL: "https://git.app.uib.no/student/exercise1-base.git",
	}
	assert.NoError(t, db.Create(sub).Error)

	grd := &submission.Grade{
		SubmissionID:       sub.ID,
		Score:              98.5,
		Feedback:           "Excellent test cases and structure",
		IsVisibleToStudent: true,
		GradedByID:         &teacher.ID,
	}
	assert.NoError(t, db.Create(grd).Error)

	// Preload bidirectional for Submission & Grade
	var foundSub submission.Submission
	err = db.Preload("Grade.GradedBy").First(&foundSub, sub.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundSub.Grade)
	assert.Equal(t, 98.5, foundSub.Grade.Score)
	assert.NotNil(t, foundSub.Grade.GradedBy)
	assert.Equal(t, "teacher@hvl.no", foundSub.Grade.GradedBy.Email)

	var foundGrade submission.Grade
	err = db.Preload("Submission.Content").First(&foundGrade, grd.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, foundGrade.Submission)
	assert.Equal(t, "Exercise 1", foundGrade.Submission.Content.Title)
}
