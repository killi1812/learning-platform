package grade_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/grade"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGradeModel_IndividualAndGroupRelations(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Group{},
		&course.Module{},
		&content.Content{},
		&content.Assignment{},
		&submission.Submission{},
		&submission.GroupSubmission{},
		&submission.IndividualSubmission{},
		&grade.Grade{},
	)

	teacher := &user.User{
		Uuid:         uuid.New(),
		Username:     "teacher_m",
		Email:        "teacher_m@example.com",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_m",
		Email:        "student_m@example.com",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	require.NoError(t, db.Create(teacher).Error)
	require.NoError(t, db.Create(student).Error)

	crs := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	require.NoError(t, db.Create(crs).Error)

	grp := &course.Group{CourseID: crs.ID, Name: "Team Alpha"}
	require.NoError(t, db.Create(grp).Error)

	mod := &course.Module{CourseID: crs.ID, Title: "Week 1", Position: 1}
	require.NoError(t, db.Create(mod).Error)

	cnt := &content.Content{ModuleID: mod.ID, Title: "Exercise 1", Position: 1, Type: content.TypeAssignment}
	require.NoError(t, db.Create(cnt).Error)

	now := time.Now()
	indivSub := &submission.IndividualSubmission{
		Submission: submission.Submission{ContentID: cnt.ID, Attempt: 1, SubmittedAt: now},
		UserID:     &student.ID,
		UserEmail:  student.Email,
	}
	require.NoError(t, db.Create(indivSub).Error)

	grpSub := &submission.GroupSubmission{
		Submission: submission.Submission{ContentID: cnt.ID, Attempt: 1, SubmittedAt: now},
		GroupID:    &grp.ID,
		UserID:     &student.ID,
	}
	require.NoError(t, db.Create(grpSub).Error)

	indGrade := &grade.Grade{
		IndividualSubmissionID: &indivSub.ID,
		Score:                  98.5,
		Feedback:               "Excellent test cases",
		IsVisibleToStudent:     true,
		GradedByID:             &teacher.ID,
	}
	require.NoError(t, db.Create(indGrade).Error)

	grpGrade := &grade.Grade{
		GroupSubmissionID:  &grpSub.ID,
		Score:              91.0,
		Feedback:           "Solid team project",
		IsVisibleToStudent: true,
		GradedByID:         &teacher.ID,
	}
	require.NoError(t, db.Create(grpGrade).Error)

	var loadedInd grade.Grade
	require.NoError(t, db.Preload("IndividualSubmission.Content").Preload("GradedBy").First(&loadedInd, indGrade.ID).Error)
	assert.Equal(t, 98.5, loadedInd.Score)
	require.NotNil(t, loadedInd.IndividualSubmission)
	require.NotNil(t, loadedInd.IndividualSubmission.Content)
	assert.Equal(t, "Exercise 1", loadedInd.IndividualSubmission.Content.Title)
	require.NotNil(t, loadedInd.GradedBy)
	assert.Equal(t, "teacher_m@example.com", loadedInd.GradedBy.Email)

	var loadedGrp grade.Grade
	require.NoError(t, db.Preload("GroupSubmission.Group").Preload("GradedBy").First(&loadedGrp, grpGrade.ID).Error)
	assert.Equal(t, 91.0, loadedGrp.Score)
	require.NotNil(t, loadedGrp.GroupSubmission)
	require.NotNil(t, loadedGrp.GroupSubmission.Group)
	assert.Equal(t, "Team Alpha", loadedGrp.GroupSubmission.Group.Name)
}
