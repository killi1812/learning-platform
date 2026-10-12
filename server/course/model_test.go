package course_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
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
		Role:         auth.ROLE_USER,
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
		CourseID: c.ID,
		Name:     "Group 1",
		Members:  []user.User{*u},
	}
	assert.NoError(t, db.Create(grp).Error)

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
	assert.Len(t, retrieved.Groups, 1)
	assert.Len(t, retrieved.Groups[0].Members, 1)
	assert.Len(t, retrieved.Modules, 2)
}
