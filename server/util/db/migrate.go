package db

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// HandleMigrations orchestrates GORM schema auto-migrations.
// In development, schemas are auto-migrated automatically on startup.
// In production, migrations execute only if runMigration (--migrate / -m) is specified.
// If runMigration is false in production and tables are missing, a warning is logged.
func HandleMigrations(db *gorm.DB, isDev bool, runMigration bool, models ...any) error {
	silentDb := db.Session(&gorm.Session{
		Logger: db.Logger.LogMode(logger.Silent),
	})

	if isDev {
		if err := silentDb.AutoMigrate(models...); err != nil {
			zap.S().Warnf("AutoMigrate failed: %+v", err)
			return err
		}
		return nil
	}

	// Production build
	if runMigration {
		if err := silentDb.AutoMigrate(models...); err != nil {
			zap.S().Errorf("Database migration failed: %+v", err)
			return err
		}
		return nil
	}

	// In production without --migrate flag: check if schema exists
	for _, model := range models {
		if !silentDb.Migrator().HasTable(model) {
			zap.S().Warn("Database schema may be outdated or uninitialized. Run with --migrate or -m to apply migrations.")
			break
		}
	}
	return nil
}
