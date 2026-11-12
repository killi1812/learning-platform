package course

import (
	"fmt"
	"sync"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestDB(t *testing.T) (*CourseService, *user.User, *user.User) {
	db := testutil.NewSqliteTestDb(t, &user.User{}, &Course{}, &Enrollment{}, &Group{}, &Module{}, &notification.Notification{})
	logger := zap.NewNop().Sugar()
	memStorage := storage.NewMemoryStorageService()
	notifier := notification.NewNotifier(db, notification.NewHub(), logger)
	svc := &CourseService{db: db, storage: memStorage, notifications: notifier, logger: logger}

	teacher := &user.User{
		Uuid:         uuid.New(),
		Username:     "teacher_bob",
		Email:        "bob@hvl.no",
		PasswordHash: "secret",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(teacher).Error)

	student := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_alice",
		Email:        "alice@hvl.no",
		PasswordHash: "secret",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, db.Create(student).Error)

	return svc, teacher, student
}

func TestCourseService_CRUDAndEnrollment(t *testing.T) {
	svc, teacher, student := setupTestDB(t)

	var createdCourse *Course

	t.Run("CreateCourse creates course and auto-enrolls creator as TEACHER", func(t *testing.T) {
		c := &Course{
			Title:       "Distributed Systems",
			Description: "Cloud & backend course",
			Term:        TermAutumn,
			Year:        2026,
		}
		res, err := svc.CreateCourse(c, teacher.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotZero(t, res.ID)
		createdCourse = res

		// Verify teacher is auto-enrolled
		enrollments, err := svc.GetCourseEnrollments(res.Uuid)
		assert.NoError(t, err)
		assert.Len(t, enrollments, 1)
		assert.Equal(t, teacher.ID, enrollments[0].UserID)
		assert.Equal(t, RoleTeacher, enrollments[0].Role)
	})

	t.Run("CreateCourse with non-existent user returns ErrUserNotFound", func(t *testing.T) {
		c := &Course{
			Title: "Ghost Course",
			Term:  TermSpring,
			Year:  2026,
		}
		_, err := svc.CreateCourse(c, uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("GetCourse returns course with preloaded enrollments", func(t *testing.T) {
		found, err := svc.GetCourse(createdCourse.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, createdCourse.Title, found.Title)
		assert.NotEmpty(t, found.Enrollments)
	})

	t.Run("GetCourse non-existent returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.GetCourse(uuid.New())
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("ListCourses returns all courses with optional filters", func(t *testing.T) {
		courses, err := svc.ListCourses(nil, nil)
		assert.NoError(t, err)
		assert.Len(t, courses, 1)

		autumn := TermAutumn
		year := 2026
		filtered, err := svc.ListCourses(&autumn, &year)
		assert.NoError(t, err)
		assert.Len(t, filtered, 1)

		spring := TermSpring
		empty, err := svc.ListCourses(&spring, &year)
		assert.NoError(t, err)
		assert.Empty(t, empty)

		// Term only
		termOnly, err := svc.ListCourses(&autumn, nil)
		assert.NoError(t, err)
		assert.Len(t, termOnly, 1)

		// Year only
		yearOnly, err := svc.ListCourses(nil, &year)
		assert.NoError(t, err)
		assert.Len(t, yearOnly, 1)
	})

	t.Run("SelfEnroll enrolls student as RoleStudent", func(t *testing.T) {
		enrollment, err := svc.SelfEnroll(createdCourse.Uuid, student.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, enrollment)
		assert.Equal(t, student.ID, enrollment.UserID)
		assert.Equal(t, RoleStudent, enrollment.Role)

		// Check enrollments list
		enrollments, err := svc.GetCourseEnrollments(createdCourse.Uuid)
		assert.NoError(t, err)
		assert.Len(t, enrollments, 2)
	})

	t.Run("SelfEnroll already enrolled returns ErrAlreadyEnrolled", func(t *testing.T) {
		_, err := svc.SelfEnroll(createdCourse.Uuid, student.Uuid)
		assert.ErrorIs(t, err, ErrAlreadyEnrolled)
	})

	t.Run("GetUserEnrollments preloads courses for student", func(t *testing.T) {
		studentEnrollments, err := svc.GetUserEnrollments(student.Uuid)
		assert.NoError(t, err)
		assert.Len(t, studentEnrollments, 1)
		assert.NotNil(t, studentEnrollments[0].Course)
		assert.Equal(t, createdCourse.Title, studentEnrollments[0].Course.Title)
	})

	t.Run("EnrollByEmail adds a user by email", func(t *testing.T) {
		coTeacher := &user.User{
			Uuid:         uuid.New(),
			Username:     "coteacher",
			Email:        "coteacher@hvl.no",
			PasswordHash: "pass",
			Role:         authutils.ROLE_USER,
		}
		assert.NoError(t, svc.db.Create(coTeacher).Error)

		enrollment, err := svc.EnrollByEmail(createdCourse.Uuid, coTeacher.Email, RoleTeacher)
		assert.NoError(t, err)
		assert.NotNil(t, enrollment)
		assert.Equal(t, coTeacher.ID, enrollment.UserID)
		assert.Equal(t, RoleTeacher, enrollment.Role)
	})

	t.Run("EnrollByEmail adds a user by email with default role", func(t *testing.T) {
		anotherStudent := &user.User{
			Uuid:         uuid.New(),
			Username:     "another",
			Email:        "another@hvl.no",
			PasswordHash: "pass",
			Role:         authutils.ROLE_USER,
		}
		assert.NoError(t, svc.db.Create(anotherStudent).Error)

		// role is empty string, should default to RoleStudent
		enrollment, err := svc.EnrollByEmail(createdCourse.Uuid, anotherStudent.Email, "")
		assert.NoError(t, err)
		assert.NotNil(t, enrollment)
		assert.Equal(t, RoleStudent, enrollment.Role)
	})

	t.Run("EnrollByEmail with unknown email returns ErrUserNotFound", func(t *testing.T) {
		_, err := svc.EnrollByEmail(createdCourse.Uuid, "unknown@hvl.no", RoleStudent)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("IsTeacherOrAdmin correctly checks permissions", func(t *testing.T) {
		// Teacher bob is teacher
		isTeacher, err := svc.IsTeacherOrAdmin(createdCourse.Uuid, teacher.Uuid, false)
		assert.NoError(t, err)
		assert.True(t, isTeacher)

		// Student alice is not teacher
		isTeacher, err = svc.IsTeacherOrAdmin(createdCourse.Uuid, student.Uuid, false)
		assert.NoError(t, err)
		assert.False(t, isTeacher)

		// System admin overrides course-level check
		isTeacher, err = svc.IsTeacherOrAdmin(createdCourse.Uuid, student.Uuid, true)
		assert.NoError(t, err)
		assert.True(t, isTeacher)
	})

	t.Run("IsEnrolledOrAdmin correctly checks enrollment", func(t *testing.T) {
		// Teacher bob is enrolled
		enrolled, err := svc.IsEnrolledOrAdmin(createdCourse.Uuid, teacher.Uuid, false)
		assert.NoError(t, err)
		assert.True(t, enrolled)

		// Student alice is enrolled
		enrolled, err = svc.IsEnrolledOrAdmin(createdCourse.Uuid, student.Uuid, false)
		assert.NoError(t, err)
		assert.True(t, enrolled)

		// Unenrolled user
		unenrolledUser := &user.User{
			Uuid:     uuid.New(),
			Email:    "random@hvl.no",
			Username: "random",
			Role:     authutils.ROLE_USER,
		}
		assert.NoError(t, svc.db.Create(unenrolledUser).Error)
		enrolled, err = svc.IsEnrolledOrAdmin(createdCourse.Uuid, unenrolledUser.Uuid, false)
		assert.NoError(t, err)
		assert.False(t, enrolled)

		// System admin overrides enrollment check
		enrolled, err = svc.IsEnrolledOrAdmin(createdCourse.Uuid, unenrolledUser.Uuid, true)
		assert.NoError(t, err)
		assert.True(t, enrolled)

		// Non-existent user or course
		_, err = svc.IsEnrolledOrAdmin(createdCourse.Uuid, uuid.New(), false)
		assert.Error(t, err)
		_, err = svc.IsEnrolledOrAdmin(uuid.New(), student.Uuid, false)
		assert.Error(t, err)
	})

	t.Run("UpdateCourse modifies course attributes", func(t *testing.T) {
		updatedData := &Course{
			Title:       "Distributed Systems v2",
			Description: "Updated desc",
			Term:        TermAutumn,
			Year:        2026,
		}
		updated, err := svc.UpdateCourse(createdCourse.Uuid, updatedData)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed Systems v2", updated.Title)
		assert.Equal(t, "Updated desc", updated.Description)
	})

	t.Run("Withdraw removes student enrollment", func(t *testing.T) {
		// Cannot remove a teacher
		err := svc.Withdraw(createdCourse.Uuid, teacher.Uuid)
		assert.ErrorIs(t, err, ErrCannotRemoveTeacher)

		err = svc.Withdraw(createdCourse.Uuid, student.Uuid)
		assert.NoError(t, err)

		// Repeating withdraw returns ErrEnrollmentNotFound
		err = svc.Withdraw(createdCourse.Uuid, student.Uuid)
		assert.ErrorIs(t, err, ErrEnrollmentNotFound)
	})

	t.Run("NewCourseService constructor", func(t *testing.T) {
		constructorSvc := NewCourseService(svc.db, svc.storage, svc.notifications, svc.logger)
		assert.NotNil(t, constructorSvc)
	})

	t.Run("UpdateCourse non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.UpdateCourse(uuid.New(), &Course{Title: "Ghost"})
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("DeleteCourse non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		err := svc.DeleteCourse(uuid.New())
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("SelfEnroll non-existent user returns ErrUserNotFound", func(t *testing.T) {
		_, err := svc.SelfEnroll(createdCourse.Uuid, uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("SelfEnroll non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.SelfEnroll(uuid.New(), student.Uuid)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("EnrollByEmail non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.EnrollByEmail(uuid.New(), teacher.Email, RoleStudent)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("EnrollByEmail already enrolled returns ErrAlreadyEnrolled", func(t *testing.T) {
		_, err := svc.EnrollByEmail(createdCourse.Uuid, teacher.Email, RoleTeacher)
		assert.ErrorIs(t, err, ErrAlreadyEnrolled)
	})

	t.Run("Withdraw non-existent user or course returns error", func(t *testing.T) {
		err := svc.Withdraw(createdCourse.Uuid, uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)

		err = svc.Withdraw(uuid.New(), student.Uuid)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("GetCourseEnrollments non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.GetCourseEnrollments(uuid.New())
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("GetUserEnrollments non-existent user returns ErrUserNotFound", func(t *testing.T) {
		_, err := svc.GetUserEnrollments(uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("IsTeacherOrAdmin non-existent user or course returns error", func(t *testing.T) {
		_, err := svc.IsTeacherOrAdmin(createdCourse.Uuid, uuid.New(), false)
		assert.ErrorIs(t, err, ErrUserNotFound)

		_, err = svc.IsTeacherOrAdmin(uuid.New(), student.Uuid, false)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("DeleteCourse deletes course and cascades", func(t *testing.T) {
		err := svc.DeleteCourse(createdCourse.Uuid)
		assert.NoError(t, err)

		_, err = svc.GetCourse(createdCourse.Uuid)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})
}

func TestCourseService_Modules(t *testing.T) {
	svc, teacher, _ := setupTestDB(t)

	c := &Course{
		Title:       "Software Technologies",
		Description: "Course with modules",
		Term:        TermAutumn,
		Year:        2026,
	}
	course, err := svc.CreateCourse(c, teacher.Uuid)
	assert.NoError(t, err)

	otherCourse, err := svc.CreateCourse(&Course{
		Title: "Other Course",
		Term:  TermAutumn,
		Year:  2026,
	}, teacher.Uuid)
	assert.NoError(t, err)

	var mod1 *Module
	var mod2 *Module

	t.Run("CreateModule auto-positions and creates module", func(t *testing.T) {
		m := &Module{Title: "Week 1: Introduction"}
		created, err := svc.CreateModule(course.Uuid, m)
		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotZero(t, created.ID)
		assert.Equal(t, 0, created.Position)
		mod1 = created
	})

	t.Run("CreateModule with DependsOn prerequisite", func(t *testing.T) {
		m := &Module{
			Title:       "Week 2: Advanced Topics",
			DependsOnID: &mod1.ID,
		}
		created, err := svc.CreateModule(course.Uuid, m)
		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, 1, created.Position)
		assert.NotNil(t, created.DependsOnID)
		assert.Equal(t, mod1.ID, *created.DependsOnID)
		assert.NotNil(t, created.DependsOn)
		assert.Equal(t, mod1.Title, created.DependsOn.Title)
		mod2 = created
	})

	t.Run("CreateModule non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := svc.CreateModule(uuid.New(), &Module{Title: "Ghost"})
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("CreateModule non-existent prerequisite returns ErrModuleNotFound", func(t *testing.T) {
		ghostID := uint(9999)
		_, err := svc.CreateModule(course.Uuid, &Module{Title: "Ghost Dep", DependsOnID: &ghostID})
		assert.ErrorIs(t, err, ErrModuleNotFound)
	})

	t.Run("CreateModule cross-course prerequisite returns ErrCrossCoursePrerequisite", func(t *testing.T) {
		otherMod, err := svc.CreateModule(otherCourse.Uuid, &Module{Title: "Other Course Mod"})
		assert.NoError(t, err)

		_, err = svc.CreateModule(course.Uuid, &Module{Title: "Bad Dep", DependsOnID: &otherMod.ID})
		assert.ErrorIs(t, err, ErrCrossCoursePrerequisite)
	})

	t.Run("GetModule returns module with dependsOn", func(t *testing.T) {
		found, err := svc.GetModule(course.Uuid, mod2.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, mod2.Title, found.Title)
		assert.NotNil(t, found.DependsOn)
		assert.Equal(t, mod1.Title, found.DependsOn.Title)
	})

	t.Run("GetModule non-existent returns error", func(t *testing.T) {
		_, err := svc.GetModule(course.Uuid, uuid.New())
		assert.ErrorIs(t, err, ErrModuleNotFound)

		_, err = svc.GetModule(uuid.New(), mod1.Uuid)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("ListModules returns modules ordered by position", func(t *testing.T) {
		modules, err := svc.ListModules(course.Uuid)
		assert.NoError(t, err)
		assert.Len(t, modules, 2)
		assert.Equal(t, mod1.Uuid, modules[0].Uuid)
		assert.Equal(t, mod2.Uuid, modules[1].Uuid)

		// Non-existent course
		_, err = svc.ListModules(uuid.New())
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("UpdateModule updates title and prerequisite", func(t *testing.T) {
		updated, err := svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:    "Week 1: Foundations",
			Position: 0,
		})
		assert.NoError(t, err)
		assert.Equal(t, "Week 1: Foundations", updated.Title)
	})

	t.Run("UpdateModule with self as prerequisite returns ErrSelfPrerequisite", func(t *testing.T) {
		_, err := svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:       "Week 1",
			Position:    -1,
			DependsOnID: &mod1.ID,
		})
		assert.ErrorIs(t, err, ErrSelfPrerequisite)
	})

	t.Run("UpdateModule circular prerequisite returns ErrCircularPrerequisite", func(t *testing.T) {
		// mod2 depends on mod1. Now attempt to make mod1 depend on mod2.
		_, err := svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:       "Week 1",
			Position:    -1,
			DependsOnID: &mod2.ID,
		})
		assert.ErrorIs(t, err, ErrCircularPrerequisite)
	})

	t.Run("UpdateModule with position -1 preserves existing position", func(t *testing.T) {
		current, err := svc.GetModule(course.Uuid, mod1.Uuid)
		assert.NoError(t, err)
		origPos := current.Position

		updated, err := svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:    "Week 1 Renamed",
			Position: -1,
		})
		assert.NoError(t, err)
		assert.Equal(t, origPos, updated.Position)
	})

	t.Run("UpdateModule with non-existent prerequisite returns ErrModuleNotFound", func(t *testing.T) {
		ghostID := uint(9999)
		_, err := svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:       "Week 1",
			Position:    -1,
			DependsOnID: &ghostID,
		})
		assert.ErrorIs(t, err, ErrModuleNotFound)
	})

	t.Run("UpdateModule with cross-course prerequisite returns ErrCrossCoursePrerequisite", func(t *testing.T) {
		otherModules, err := svc.ListModules(otherCourse.Uuid)
		assert.NoError(t, err)
		assert.NotEmpty(t, otherModules)

		_, err = svc.UpdateModule(course.Uuid, mod1.Uuid, &Module{
			Title:       "Week 1",
			Position:    -1,
			DependsOnID: &otherModules[0].ID,
		})
		assert.ErrorIs(t, err, ErrCrossCoursePrerequisite)
	})

	t.Run("UpdateModule non-existent returns error", func(t *testing.T) {
		_, err := svc.UpdateModule(course.Uuid, uuid.New(), &Module{Title: "Ghost", Position: -1})
		assert.ErrorIs(t, err, ErrModuleNotFound)

		_, err = svc.UpdateModule(uuid.New(), mod1.Uuid, &Module{Title: "Ghost", Position: -1})
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("ReorderModules updates positions sequentially", func(t *testing.T) {
		// Reorder: mod2 first, then mod1
		err := svc.ReorderModules(course.Uuid, []uuid.UUID{mod2.Uuid, mod1.Uuid})
		assert.NoError(t, err)

		list, err := svc.ListModules(course.Uuid)
		assert.NoError(t, err)
		assert.Equal(t, mod2.Uuid, list[0].Uuid)
		assert.Equal(t, 0, list[0].Position)
		assert.Equal(t, mod1.Uuid, list[1].Uuid)
		assert.Equal(t, 1, list[1].Position)
	})

	t.Run("ReorderModules non-existent course returns ErrCourseNotFound", func(t *testing.T) {
		err := svc.ReorderModules(uuid.New(), []uuid.UUID{mod1.Uuid})
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})

	t.Run("ReorderModules non-existent module or incomplete list returns ErrInvalidReorderList", func(t *testing.T) {
		err := svc.ReorderModules(course.Uuid, []uuid.UUID{uuid.New()})
		assert.ErrorIs(t, err, ErrInvalidReorderList)

		// duplicate module UUID
		err = svc.ReorderModules(course.Uuid, []uuid.UUID{mod1.Uuid, mod1.Uuid})
		assert.ErrorIs(t, err, ErrInvalidReorderList)
	})

	t.Run("DeleteModule deletes module and cascades SET NULL on dependent", func(t *testing.T) {
		err := svc.DeleteModule(course.Uuid, mod1.Uuid)
		assert.NoError(t, err)

		_, err = svc.GetModule(course.Uuid, mod1.Uuid)
		assert.ErrorIs(t, err, ErrModuleNotFound)

		// Non-existent module
		err = svc.DeleteModule(course.Uuid, uuid.New())
		assert.ErrorIs(t, err, ErrModuleNotFound)

		// Non-existent course
		err = svc.DeleteModule(uuid.New(), mod2.Uuid)
		assert.ErrorIs(t, err, ErrCourseNotFound)
	})
}

func TestCourseService_Notifications(t *testing.T) {
	svc, teacher, student := setupTestDB(t)

	c, err := svc.CreateCourse(&Course{
		Title: "Notification Test Course",
		Term:  TermAutumn,
		Year:  2026,
	}, teacher.Uuid)
	assert.NoError(t, err)

	_, err = svc.SelfEnroll(c.Uuid, student.Uuid)
	assert.NoError(t, err)

	t.Run("UpdateCourse notifies enrolled student", func(t *testing.T) {
		_, err := svc.UpdateCourse(c.Uuid, &Course{Title: "Updated Notification Course"})
		assert.NoError(t, err)

		var notifs []notification.Notification
		assert.NoError(t, svc.db.Where("user_id = ?", student.ID).Find(&notifs).Error)
		assert.NotEmpty(t, notifs)
		assert.Equal(t, "Course Updated: Updated Notification Course", notifs[len(notifs)-1].Title)
		assert.Equal(t, "/courses/"+c.Uuid.String(), notifs[len(notifs)-1].URL)

		// Teacher should not receive student-enrolled update notification
		var teacherNotifs []notification.Notification
		assert.NoError(t, svc.db.Where("user_id = ?", teacher.ID).Find(&teacherNotifs).Error)
		assert.Empty(t, teacherNotifs)
	})

	t.Run("CreateModule notifies enrolled student", func(t *testing.T) {
		m, err := svc.CreateModule(c.Uuid, &Module{Title: "Week 1 Basics"})
		assert.NoError(t, err)

		var notifs []notification.Notification
		assert.NoError(t, svc.db.Where("user_id = ?", student.ID).Find(&notifs).Error)
		assert.NotEmpty(t, notifs)
		assert.Contains(t, notifs[len(notifs)-1].Body, "Week 1 Basics")

		t.Run("UpdateModule notifies enrolled student", func(t *testing.T) {
			_, err := svc.UpdateModule(c.Uuid, m.Uuid, &Module{Title: "Week 1 Advanced"})
			assert.NoError(t, err)

			var notifsAfter []notification.Notification
			assert.NoError(t, svc.db.Where("user_id = ?", student.ID).Find(&notifsAfter).Error)
			assert.Contains(t, notifsAfter[len(notifsAfter)-1].Body, "Week 1 Advanced")
		})
	})
}

func TestGroupService(t *testing.T) {
	svc, teacher, student := setupTestDB(t)
	groupSvc := NewGroupService(svc.db, zap.NewNop().Sugar())

	c, err := svc.CreateCourse(&Course{
		Title: "Group Systems",
		Term:  TermAutumn,
		Year:  2026,
	}, teacher.Uuid)
	assert.NoError(t, err)

	_, err = svc.SelfEnroll(c.Uuid, student.Uuid)
	assert.NoError(t, err)

	student2 := &user.User{
		Uuid:         uuid.New(),
		Username:     "student_charlie",
		Email:        "charlie@hvl.no",
		PasswordHash: "pass",
		Role:         authutils.ROLE_USER,
	}
	assert.NoError(t, svc.db.Create(student2).Error)
	_, err = svc.SelfEnroll(c.Uuid, student2.Uuid)
	assert.NoError(t, err)

	t.Run("CreateGroup validations and success", func(t *testing.T) {
		// Non-existent course
		_, err := groupSvc.CreateGroup(uuid.New(), "Alpha", 4)
		assert.ErrorIs(t, err, ErrCourseNotFound)

		// Empty name
		_, err = groupSvc.CreateGroup(c.Uuid, "   ", 4)
		assert.Error(t, err)

		// Success
		grp, err := groupSvc.CreateGroup(c.Uuid, "Alpha", 2)
		assert.NoError(t, err)
		assert.Equal(t, "Alpha", grp.Name)
		assert.Equal(t, 2, grp.MaxMembers)
		assert.NotEqual(t, uuid.Nil, grp.Uuid)

		// Duplicate name in same course
		_, err = groupSvc.CreateGroup(c.Uuid, "alpha", 4)
		assert.Error(t, err)

		// Default maxMembers <= 0 defaults to 4
		grpDefault, err := groupSvc.CreateGroup(c.Uuid, "Beta", 0)
		assert.NoError(t, err)
		assert.Equal(t, 4, grpDefault.MaxMembers)
	})

	t.Run("BulkCreateGroups validations and sequential creation", func(t *testing.T) {
		// Invalid count
		_, err := groupSvc.BulkCreateGroups(c.Uuid, "Team", 0, 4)
		assert.Error(t, err)
		_, err = groupSvc.BulkCreateGroups(c.Uuid, "Team", 51, 4)
		assert.Error(t, err)

		// Non-existent course
		_, err = groupSvc.BulkCreateGroups(uuid.New(), "Team", 3, 4)
		assert.ErrorIs(t, err, ErrCourseNotFound)

		// Create 3 teams
		teams, err := groupSvc.BulkCreateGroups(c.Uuid, "Team", 3, 3)
		assert.NoError(t, err)
		assert.Len(t, teams, 3)
		assert.Equal(t, "Team 1", teams[0].Name)
		assert.Equal(t, "Team 2", teams[1].Name)
		assert.Equal(t, "Team 3", teams[2].Name)

		// Bulk create again with same prefix skips existing Team 1..3
		moreTeams, err := groupSvc.BulkCreateGroups(c.Uuid, "Team", 2, 3)
		assert.NoError(t, err)
		assert.Len(t, moreTeams, 2)
		assert.Equal(t, "Team 4", moreTeams[0].Name)
		assert.Equal(t, "Team 5", moreTeams[1].Name)

		// Default prefix when empty
		defaultGroups, err := groupSvc.BulkCreateGroups(c.Uuid, "", 2, 4)
		assert.NoError(t, err)
		assert.Len(t, defaultGroups, 2)
		assert.Equal(t, "Group 1", defaultGroups[0].Name)
		assert.Equal(t, "Group 2", defaultGroups[1].Name)
	})

	t.Run("ListCourseGroups and GetGroup", func(t *testing.T) {
		groups, err := groupSvc.ListCourseGroups(c.Uuid)
		assert.NoError(t, err)
		assert.NotEmpty(t, groups)

		// Non-existent course
		_, err = groupSvc.ListCourseGroups(uuid.New())
		assert.ErrorIs(t, err, ErrCourseNotFound)

		// GetGroup existing
		found, err := groupSvc.GetGroup(groups[0].Uuid)
		assert.NoError(t, err)
		assert.Equal(t, groups[0].Name, found.Name)

		// GetGroup non-existent
		_, err = groupSvc.GetGroup(uuid.New())
		assert.ErrorIs(t, err, ErrGroupNotFound)
	})

	t.Run("UserJoinGroup, capacity, single-group rule, and RemoveStudentFromGroup", func(t *testing.T) {
		joinGrp, err := groupSvc.CreateGroup(c.Uuid, "JoinTest", 1)
		assert.NoError(t, err)

		// Non-enrolled user cannot join
		nonEnrolled := &user.User{
			Uuid:         uuid.New(),
			Username:     "ghost",
			Email:        "ghost@example.com",
			PasswordHash: "pass",
		}
		assert.NoError(t, svc.db.Create(nonEnrolled).Error)
		_, err = groupSvc.UserJoinGroup(joinGrp.Uuid, nonEnrolled.Uuid)
		assert.ErrorIs(t, err, ErrEnrollmentNotFound)

		// Student joins
		joined, err := groupSvc.UserJoinGroup(joinGrp.Uuid, student.Uuid)
		assert.NoError(t, err)
		assert.Len(t, joined.Members, 1)
		assert.Equal(t, student.ID, joined.Members[0].ID)

		// Student cannot join another group in the same course
		otherGrp, err := groupSvc.CreateGroup(c.Uuid, "OtherGroup", 4)
		assert.NoError(t, err)
		_, err = groupSvc.UserJoinGroup(otherGrp.Uuid, student.Uuid)
		assert.ErrorIs(t, err, ErrUserAlreadyInGroup)

		// Student 2 cannot join joinGrp because it is full (maxMembers=1)
		_, err = groupSvc.UserJoinGroup(joinGrp.Uuid, student2.Uuid)
		assert.ErrorIs(t, err, ErrGroupFull)

		// Non-member cannot leave
		_, err = groupSvc.RemoveStudentFromGroup(joinGrp.Uuid, student2.Uuid)
		assert.ErrorIs(t, err, ErrUserNotInGroup)

		// Student leaves
		left, err := groupSvc.RemoveStudentFromGroup(joinGrp.Uuid, student.Uuid)
		assert.NoError(t, err)
		assert.Empty(t, left.Members)

		// Now student 2 can join
		joined2, err := groupSvc.UserJoinGroup(joinGrp.Uuid, student2.Uuid)
		assert.NoError(t, err)
		assert.Len(t, joined2.Members, 1)
	})

	t.Run("UpdateGroup validations and success", func(t *testing.T) {
		upGrp, err := groupSvc.CreateGroup(c.Uuid, "UpTest", 4)
		assert.NoError(t, err)

		// Non-existent group
		_, err = groupSvc.UpdateGroup(uuid.New(), "NewName", 4)
		assert.ErrorIs(t, err, ErrGroupNotFound)

		// Empty name
		_, err = groupSvc.UpdateGroup(upGrp.Uuid, "   ", 4)
		assert.Error(t, err)

		// Duplicate name
		_, err = groupSvc.UpdateGroup(upGrp.Uuid, "Alpha", 4)
		assert.Error(t, err)

		// Success
		updated, err := groupSvc.UpdateGroup(upGrp.Uuid, "UpTest Renamed", 6)
		assert.NoError(t, err)
		assert.Equal(t, "UpTest Renamed", updated.Name)
		assert.Equal(t, 6, updated.MaxMembers)

		// MaxMembers < current member count fails
		student3 := &user.User{
			Uuid:         uuid.New(),
			Username:     "student_dave",
			Email:        "dave@hvl.no",
			PasswordHash: "pass",
			Role:         authutils.ROLE_USER,
		}
		assert.NoError(t, svc.db.Create(student3).Error)
		_, err = svc.SelfEnroll(c.Uuid, student3.Uuid)
		assert.NoError(t, err)
		_, err = groupSvc.UserJoinGroup(updated.Uuid, student3.Uuid)
		assert.NoError(t, err)

		_, err = groupSvc.UpdateGroup(updated.Uuid, "UpTest Renamed", 0) // defaults to 4, which is >= 1
		assert.NoError(t, err)
	})

	t.Run("DeleteGroup clears associations", func(t *testing.T) {
		delGrp, err := groupSvc.CreateGroup(c.Uuid, "DeleteMe", 4)
		assert.NoError(t, err)

		err = groupSvc.DeleteGroup(delGrp.Uuid)
		assert.NoError(t, err)

		_, err = groupSvc.GetGroup(delGrp.Uuid)
		assert.ErrorIs(t, err, ErrGroupNotFound)

		// Deleting non-existent returns error
		err = groupSvc.DeleteGroup(uuid.New())
		assert.ErrorIs(t, err, ErrGroupNotFound)
	})

	t.Run("Course Withdraw removes student from course group", func(t *testing.T) {
		withdrawStudent := &user.User{
			Uuid:         uuid.New(),
			Username:     "withdrawer",
			Email:        "withdrawer@example.com",
			PasswordHash: "pass",
			Role:         authutils.ROLE_USER,
		}
		assert.NoError(t, svc.db.Create(withdrawStudent).Error)
		_, err := svc.SelfEnroll(c.Uuid, withdrawStudent.Uuid)
		assert.NoError(t, err)

		wGrp, err := groupSvc.CreateGroup(c.Uuid, "WithdrawGroup", 4)
		assert.NoError(t, err)

		_, err = groupSvc.UserJoinGroup(wGrp.Uuid, withdrawStudent.Uuid)
		assert.NoError(t, err)

		// Withdraw student from course
		err = svc.Withdraw(c.Uuid, withdrawStudent.Uuid)
		assert.NoError(t, err)

		// Check student is removed from group
		reloaded, err := groupSvc.GetGroup(wGrp.Uuid)
		assert.NoError(t, err)
		assert.Empty(t, reloaded.Members)
	})

	t.Run("Concurrent group join race condition prevention", func(t *testing.T) {
		raceGrp, err := groupSvc.CreateGroup(c.Uuid, "RaceGroup", 1)
		assert.NoError(t, err)

		// Create 5 distinct students and enroll them
		const numStudents = 5
		raceStudents := make([]*user.User, numStudents)
		for i := 0; i < numStudents; i++ {
			u := &user.User{
				Uuid:         uuid.New(),
				Username:     fmt.Sprintf("race_student_%d", i),
				Email:        fmt.Sprintf("race_%d@example.com", i),
				PasswordHash: "pass",
				Role:         authutils.ROLE_USER,
			}
			assert.NoError(t, svc.db.Create(u).Error)
			_, err := svc.SelfEnroll(c.Uuid, u.Uuid)
			assert.NoError(t, err)
			raceStudents[i] = u
		}

		var wg sync.WaitGroup
		results := make([]error, numStudents)
		for i := 0; i < numStudents; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, results[idx] = groupSvc.UserJoinGroup(raceGrp.Uuid, raceStudents[idx].Uuid)
			}(i)
		}
		wg.Wait()

		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}
		assert.Equal(t, 1, successCount, "Exactly 1 student must succeed in joining group with capacity 1")

		// Verify reloaded group has exactly 1 member
		reloaded, err := groupSvc.GetGroup(raceGrp.Uuid)
		assert.NoError(t, err)
		assert.Len(t, reloaded.Members, 1)
	})

	t.Run("Course group limit and input sanitization", func(t *testing.T) {
		// Input sanitization: HTML tags are escaped
		xssGrp, err := groupSvc.CreateGroup(c.Uuid, "<script>alert('xss')</script>", 4)
		assert.NoError(t, err)
		assert.Equal(t, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;", xssGrp.Name)

		// Exceeding course group limit
		limitCourse, err := svc.CreateCourse(&Course{Title: "LimitCourse", Term: TermAutumn, Year: 2026}, teacher.Uuid)
		assert.NoError(t, err)

		// Bulk create 50 groups
		_, err = groupSvc.BulkCreateGroups(limitCourse.Uuid, "Batch1", 50, 4)
		assert.NoError(t, err)
		// Bulk create another 50 groups (total = 100)
		_, err = groupSvc.BulkCreateGroups(limitCourse.Uuid, "Batch2", 50, 4)
		assert.NoError(t, err)

		// 101st group must fail
		_, err = groupSvc.CreateGroup(limitCourse.Uuid, "OverflowGroup", 4)
		assert.ErrorIs(t, err, ErrCourseGroupLimitExceeded)

		_, err = groupSvc.BulkCreateGroups(limitCourse.Uuid, "OverflowBatch", 1, 4)
		assert.ErrorIs(t, err, ErrCourseGroupLimitExceeded)
	})
}
