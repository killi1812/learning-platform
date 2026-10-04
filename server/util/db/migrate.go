package db

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HandleMigrations orchestrates GORM schema auto-migrations.
// In development, schemas are auto-migrated automatically on startup.
// In production, migrations execute only if runMigration (--migrate / -m) is specified.
// If runMigration is false in production and tables are missing, a warning is logged.
func HandleMigrations(db *gorm.DB, isDev bool, runMigration bool, models ...any) error {
	if isDev {
		zap.S().Info("Auto-migrating database schema in development mode...")
		if err := db.AutoMigrate(models...); err != nil {
			zap.S().Warnf("AutoMigrate failed: %+v", err)
			return err
		}
		zap.S().Info("Database AutoMigrate complete.")
		return nil
	}

	// Production build
	if runMigration {
		zap.S().Info("Applying database migrations (--migrate flag detected)...")
		if err := db.AutoMigrate(models...); err != nil {
			zap.S().Errorf("Database migration failed: %+v", err)
			return err
		}
		zap.S().Info("Database migrations applied successfully.")
		return nil
	}

	// In production without --migrate flag: check if schema exists
	for _, model := range models {
		if !db.Migrator().HasTable(model) {
			zap.S().Warn("Database schema may be outdated or uninitialized. Run with --migrate or -m to apply migrations.")
			break
		}
	}
	return nil
}
