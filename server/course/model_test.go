package course_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCourseAggregate_CRUDAndAssociations(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Group{},
		&course.Module{},
	)

	u := &user.User{
		Uuid:         uuid.New(),
		Username:     "teacher_anna",
		Email:        "anna@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(u).Error)

	c := &course.Course{
		Title:       "DAT250: Advanced Software Technologies",
		Description: "Learning Platform Project",
		Term:        course.TermAutumn,
		Year:        2026,
	}
	assert.NoError(t, db.Create(c).Error)
	assert.NotZero(t, c.ID)
	assert.NotEqual(t, uuid.Nil, c.Uuid)

	// Update
	cUpdate := &course.Course{
		Title:       "DAT250: Modern Software Technologies",
		Description: "Updated description",
		Term:        course.TermSpring,
		Year:        2027,
	}
	c.Update(cUpdate)
	assert.NoError(t, db.Save(c).Error)

	// Enrollment
	enr := &course.Enrollment{
		UserID:     u.ID,
		UserEmail:  u.Email,
		CourseID:   c.ID,
		Role:       course.RoleTeacher,
		EnrolledAt: time.Now(),
	}
	assert.NoError(t, db.Create(enr).Error)

	// Group
	grp := &course.Group{
		CourseID:   c.ID,
		Name:       "Group 1",
		MaxMembers: 3,
		Members:    []user.User{*u},
	}
	assert.NoError(t, db.Create(grp).Error)
	assert.NotEqual(t, uuid.Nil, grp.Uuid)
	assert.Equal(t, 3, grp.MaxMembers)

	// Group BeforeCreate default maxMembers
	grp2 := &course.Group{
		CourseID: c.ID,
		Name:     "Group 2",
	}
	assert.NoError(t, db.Create(grp2).Error)
	assert.NotEqual(t, uuid.Nil, grp2.Uuid)
	assert.Equal(t, 4, grp2.MaxMembers)

	// Modules with ordering and dependency
	m1 := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m1).Error)

	m2 := &course.Module{CourseID: c.ID, Title: "Week 2", Position: 2, DependsOnID: &m1.ID}
	assert.NoError(t, db.Create(m2).Error)

	// Preload all associations from Course
	var retrieved course.Course
	err := db.
		Preload("Enrollments.User").
		Preload("Groups.Members").
		Preload("Modules.DependsOn").
		First(&retrieved, c.ID).Error

	assert.NoError(t, err)
	assert.Equal(t, "DAT250: Modern Software Technologies", retrieved.Title)
	assert.Len(t, retrieved.Enrollments, 1)
	assert.Equal(t, course.RoleTeacher, retrieved.Enrollments[0].Role)
	assert.Len(t, retrieved.Groups, 2)
	assert.Len(t, retrieved.Groups[0].Members, 1)
	assert.Len(t, retrieved.Modules, 2)

	// Verify CourseDto and EnrollmentDto mapping with teacher usernames and email fallback
	retrieved.Enrollments = append(retrieved.Enrollments, course.Enrollment{
		Role:      course.RoleTeacher,
		UserEmail: "fallback@hvl.no",
	})
	courseDto := course.CourseDto{}.FromModel(&retrieved)
	assert.Equal(t, []string{"teacher_anna", "fallback@hvl.no"}, courseDto.Teachers)

	enrDto := course.EnrollmentDto{}.FromModel(&retrieved.Enrollments[0])
	assert.Equal(t, "teacher_anna", enrDto.Username)
}

func TestGradingScale_ValidationAndCalculation(t *testing.T) {
	def := course.DefaultGradingScale()
	assert.Equal(t, 90.0, def.ScaleA)
	assert.Equal(t, 80.0, def.ScaleB)
	assert.Equal(t, 60.0, def.ScaleC)
	assert.Equal(t, 50.0, def.ScaleD)
	assert.Equal(t, 40.0, def.ScaleE)
	assert.NoError(t, def.Validate())

	// Valid calculations
	assert.Equal(t, "A", def.CalculateGrade(100.0))
	assert.Equal(t, "A", def.CalculateGrade(90.0))
	assert.Equal(t, "B", def.CalculateGrade(89.9))
	assert.Equal(t, "B", def.CalculateGrade(80.0))
	assert.Equal(t, "C", def.CalculateGrade(79.9))
	assert.Equal(t, "C", def.CalculateGrade(60.0))
	assert.Equal(t, "D", def.CalculateGrade(59.9))
	assert.Equal(t, "D", def.CalculateGrade(50.0))
	assert.Equal(t, "E", def.CalculateGrade(49.9))
	assert.Equal(t, "E", def.CalculateGrade(40.0))
	assert.Equal(t, "F", def.CalculateGrade(39.9))
	assert.Equal(t, "F", def.CalculateGrade(0.0))

	// Invalid scales
	invalidCases := []course.GradingScale{
		{ScaleA: 105, ScaleB: 80, ScaleC: 60, ScaleD: 50, ScaleE: 40},
		{ScaleA: 80, ScaleB: 80, ScaleC: 60, ScaleD: 50, ScaleE: 40},
		{ScaleA: 90, ScaleB: 50, ScaleC: 60, ScaleD: 50, ScaleE: 40},
		{ScaleA: 90, ScaleB: 80, ScaleC: 45, ScaleD: 50, ScaleE: 40},
		{ScaleA: 90, ScaleB: 80, ScaleC: 60, ScaleD: 35, ScaleE: 40},
		{ScaleA: 90, ScaleB: 80, ScaleC: 60, ScaleD: 50, ScaleE: 0},
		{ScaleA: 90, ScaleB: 80, ScaleC: 60, ScaleD: 50, ScaleE: -5},
	}
	for _, tc := range invalidCases {
		assert.ErrorIs(t, tc.Validate(), course.ErrInvalidGradingScale)
	}

	c := &course.Course{GradingScale: def}
	assert.Equal(t, "A", c.CalculateGrade(95.0))
	assert.Equal(t, "F", c.CalculateGrade(20.0))
}
