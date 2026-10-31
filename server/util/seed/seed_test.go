package seed

import (
	"os"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSeed(t *testing.T) {
	origBuild := app.Build
	app.Build = app.BuildProd
	defer func() { app.Build = origBuild }()

	db := testutil.NewSqliteTestDb(t, &user.User{})
	app.Test()
	app.Provide(func() *gorm.DB { return db })
	app.Provide(zap.S)
	app.Provide(user.NewUserCrudService)

	origPass := os.Getenv(_PASSWORD_ENV)
	defer os.Setenv(_PASSWORD_ENV, origPass)

	t.Run("Empty password env returns error", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "")
		err := createSuperAdmin()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "env variable is empty")
	})

	t.Run("Short password returns error", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "short")
		err := createSuperAdmin()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})

	t.Run("Valid password creates superadmin and Insert succeeds", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "superpassword123")
		err := createSuperAdmin()
		assert.NoError(t, err)

		// Running again detects existing superadmin and skips
		err = createSuperAdmin()
		assert.NoError(t, err)

		// Insert() wrapper runs without panic
		assert.NotPanics(t, func() {
			Insert()
		})
	})

	t.Run("DB error on GetUserByEmail returns error", func(t *testing.T) {
		closedDb := testutil.NewSqliteTestDb(t, &user.User{})
		sqlDB, err := closedDb.DB()
		assert.NoError(t, err)
		_ = sqlDB.Close()

		app.Test()
		app.Provide(func() *gorm.DB { return closedDb })
		app.Provide(zap.S)
		app.Provide(user.NewUserCrudService)

		err = createSuperAdmin()
		assert.Error(t, err)
	})

	t.Run("Insert handles error gracefully when createSuperAdmin fails", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "")
		assert.NotPanics(t, func() {
			Insert()
		})
	})
}

func TestSeedInsert(t *testing.T) {
	origBuild := app.Build
	defer func() { app.Build = origBuild }()

	origPass := os.Getenv(_PASSWORD_ENV)
	defer os.Setenv(_PASSWORD_ENV, origPass)
	os.Setenv(_PASSWORD_ENV, "superpassword123")

	t.Run("Prod build runs superadmin and skips dev entities", func(t *testing.T) {
		app.Build = app.BuildProd

		db := testutil.NewSqliteTestDb(t,
			&user.User{},
			&course.Course{},
			&course.Enrollment{},
			&course.Module{},
			&content.Content{},
			&content.Assignment{},
			&content.File{},
		)

		app.Test()
		app.Provide(func() *gorm.DB { return db })
		app.Provide(zap.S)
		app.Provide(func() storage.IStorageService { return storage.NewMemoryStorageService() })
		app.Provide(user.NewUserCrudService)
		app.Provide(course.NewCourseService)
		app.Provide(content.NewContentService)

		assert.NotPanics(t, func() {
			Insert()
		})

		var userCount int64
		db.Model(&user.User{}).Count(&userCount)
		assert.Equal(t, int64(1), userCount) // superadmin only

		var courseCount int64
		db.Model(&course.Course{}).Count(&courseCount)
		assert.Equal(t, int64(0), courseCount)
	})

	t.Run("Dev build seeds superadmin, teachers, students, courses, modules, and assignments", func(t *testing.T) {
		app.Build = app.BuildDev

		db := testutil.NewSqliteTestDb(t,
			&user.User{},
			&course.Course{},
			&course.Enrollment{},
			&course.Module{},
			&content.Content{},
			&content.Assignment{},
			&content.File{},
		)

		app.Test()
		app.Provide(func() *gorm.DB { return db })
		app.Provide(zap.S)
		app.Provide(func() storage.IStorageService { return storage.NewMemoryStorageService() })
		app.Provide(user.NewUserCrudService)
		app.Provide(course.NewCourseService)
		app.Provide(content.NewContentService)

		assert.NotPanics(t, func() {
			Insert()
		})

		var userCount int64
		db.Model(&user.User{}).Count(&userCount)
		assert.Equal(t, int64(14), userCount) // superadmin + 3 teachers + 10 students

		// Verify all teacher and student emails are @example.com
		var users []user.User
		db.Find(&users)
		for _, u := range users {
			assert.Contains(t, u.Email, "@example.com")
		}

		var courseCount int64
		db.Model(&course.Course{}).Count(&courseCount)
		assert.Equal(t, int64(2), courseCount)

		var moduleCount int64
		db.Model(&course.Module{}).Count(&moduleCount)
		assert.Equal(t, int64(4), moduleCount)

		var assignmentCount int64
		db.Model(&content.Assignment{}).Count(&assignmentCount)
		assert.Equal(t, int64(4), assignmentCount)

		var fileCount int64
		db.Model(&content.File{}).Count(&fileCount)
		assert.Equal(t, int64(4), fileCount)

		var enrollmentCount int64
		db.Model(&course.Enrollment{}).Count(&enrollmentCount)
		assert.Equal(t, int64(18), enrollmentCount) // C1: 2 teachers + 10 students; C2: 1 teacher + 5 students

		// Second run is idempotent and does not duplicate
		assert.NotPanics(t, func() {
			Insert()
		})

		var userCountAfter int64
		db.Model(&user.User{}).Count(&userCountAfter)
		assert.Equal(t, int64(14), userCountAfter)
	})

	t.Run("Focused seed functions return seeded items for downstream use", func(t *testing.T) {
		db := testutil.NewSqliteTestDb(t,
			&user.User{},
			&course.Course{},
			&course.Enrollment{},
			&course.Module{},
			&content.Content{},
			&content.Assignment{},
			&content.File{},
		)

		app.Test()
		app.Provide(func() *gorm.DB { return db })
		app.Provide(zap.S)
		app.Provide(func() storage.IStorageService { return storage.NewMemoryStorageService() })
		app.Provide(user.NewUserCrudService)
		app.Provide(course.NewCourseService)
		app.Provide(content.NewContentService)

		teachers, err := seedTeachers()
		assert.NoError(t, err)
		assert.Len(t, teachers, 3)
		assert.Equal(t, "teacher1@example.com", teachers[0].Email)

		students, err := seedStudents()
		assert.NoError(t, err)
		assert.Len(t, students, 10)
		assert.Equal(t, "student1@example.com", students[0].Email)

		courses, err := seedCourses(teachers)
		assert.NoError(t, err)
		assert.Len(t, courses, 2)

		enrollments, err := seedEnrollments(courses, teachers, students)
		assert.NoError(t, err)
		assert.NotEmpty(t, enrollments)

		modules, err := seedModules(courses)
		assert.NoError(t, err)
		assert.Len(t, modules, 4)

		assignments, err := seedAssignments(modules)
		assert.NoError(t, err)
		assert.Len(t, assignments, 4)

		files, err := seedFiles(courses, modules)
		assert.NoError(t, err)
		assert.Len(t, files, 4)

		filesAgain, err := seedFiles(courses, modules)
		assert.NoError(t, err)
		assert.Len(t, filesAgain, 4)

		// Calling seedTeachers again returns existing items without duplicating
		teachersAgain, err := seedTeachers()
		assert.NoError(t, err)
		assert.Len(t, teachersAgain, 3)
		assert.Equal(t, teachers[0].ID, teachersAgain[0].ID)
	})

	t.Run("seedTeachers and seedStudents do not fail when users already exist with old email", func(t *testing.T) {
		db := testutil.NewSqliteTestDb(t, &user.User{})
		app.Test()
		app.Provide(func() *gorm.DB { return db })
		app.Provide(zap.S)
		app.Provide(user.NewUserCrudService)

		userCrud := user.NewUserCrudService(db, zap.S())

		// Pre-create user with username "teacher1" and old email "teacher1@hvl.no"
		_, err := userCrud.Create(&user.User{
			Username: "teacher1",
			Email:    "teacher1@hvl.no",
			Role:     authutils.ROLE_USER,
		}, "Pa$$w0rd")
		assert.NoError(t, err)

		teachers, err := seedTeachers()
		assert.NoError(t, err)
		assert.Len(t, teachers, 3)
		assert.Equal(t, "teacher1@example.com", teachers[0].Email)
	})
}
