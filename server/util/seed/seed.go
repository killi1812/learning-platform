package seed

import (
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Insert runs seed functions and inserts seeded data into the database.
func Insert() {
	// Silence GORM SQL query tracing during database seeding
	var gormDb *gorm.DB
	if err := app.InvokeErr(func(db *gorm.DB) { gormDb = db }); err == nil && gormDb != nil {
		origLogger := gormDb.Logger
		gormDb.Logger = origLogger.LogMode(logger.Silent)
		defer func() {
			gormDb.Logger = origLogger
		}()
	}

	if err := createSuperAdmin(); err != nil {
		zap.S().DPanicf("Failed to create superadmin (MongoDB offline?): %+v", err)
	}

	if app.Build != app.BuildDev {
		return
	}

	var (
		userCrud       user.IUserCrudService
		courseService  course.ICourseService
		contentService content.IContentService
	)
	app.Invoke(func(u user.IUserCrudService, c course.ICourseService, cnt content.IContentService) {
		userCrud = u
		courseService = c
		contentService = cnt
	})

	if userCrud == nil || courseService == nil || contentService == nil {
		zap.S().Errorln("Required services (UserCrudService, CourseService, ContentService) not available in container")
		return
	}

	zap.S().Infoln("Seeding dev teachers, students, courses, modules, and assignments...")

	teachers, err := seedTeachers(userCrud)
	if err != nil {
		zap.S().Errorf("Failed to seed teachers: %+v", err)
		return
	}

	students, err := seedStudents(userCrud)
	if err != nil {
		zap.S().Errorf("Failed to seed students: %+v", err)
		return
	}

	courses, err := seedCourses(courseService, teachers)
	if err != nil {
		zap.S().Errorf("Failed to seed courses: %+v", err)
		return
	}

	if _, err := seedEnrollments(courseService, courses, teachers, students); err != nil {
		zap.S().Errorf("Failed to seed enrollments: %+v", err)
		return
	}

	modules, err := seedModules(courseService, courses)
	if err != nil {
		zap.S().Errorf("Failed to seed modules: %+v", err)
		return
	}

	if _, err := seedAssignments(contentService, modules); err != nil {
		zap.S().Errorf("Failed to seed assignments: %+v", err)
		return
	}

	zap.S().Infoln("Successfully seeded dev data")
}
