package seed

import (
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Insert runs seed functions and inserts seeded data into the database.
func Insert() {
	// Silence GORM SQL query tracing during database seeding
	// TODO: silence with createing a new silent connection not altering the exiting one
	var gormDb *gorm.DB
	if err := app.InvokeErr(func(db *gorm.DB) { gormDb = db }); err == nil && gormDb != nil {
		origLogger := gormDb.Logger
		gormDb.Logger = origLogger.LogMode(logger.Silent)
		defer func() {
			gormDb.Logger = origLogger
		}()
	}

	if err := createSuperAdmin(); err != nil {
		zap.S().DPanicf("Failed to create superadmin, %+v", err)
	}

	if app.Build != app.BuildDev {
		return
	}

	zap.S().Infoln("Seeding dev teachers, students, courses, modules, and assignments...")

	teachers, err := seedTeachers()
	if err != nil {
		zap.S().DPanicf("Failed to seed teachers: %+v", err)
	}

	students, err := seedStudents()
	if err != nil {
		zap.S().DPanicf("Failed to seed students: %+v", err)
	}

	courses, err := seedCourses(teachers)
	if err != nil {
		zap.S().DPanicf("Failed to seed courses: %+v", err)
	}

	if _, err := seedEnrollments(courses, teachers, students); err != nil {
		zap.S().DPanicf("Failed to seed enrollments: %+v", err)
	}

	modules, err := seedModules(courses)
	if err != nil {
		zap.S().DPanicf("Failed to seed modules: %+v", err)
	}

	if _, err := seedAssignments(modules); err != nil {
		zap.S().DPanicf("Failed to seed assignments: %+v", err)
	}

	zap.S().Infoln("Successfully seeded dev data")
}
