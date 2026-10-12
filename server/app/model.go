package app

import (
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	registeredModels []any
	migrateOnce      sync.Once
)

// RegisterModel registers one or more models for database auto-migration in app.
func RegisterModel(models ...any) {
	registeredModels = append(registeredModels, models...)
}

// Migrate sends all registered models to util/db to perform database migrations.
func Migrate() error {
	var err error
	migrateOnce.Do(func() {
		Invoke(func(db *gorm.DB) {
			err = HandleMigrations(db, RunMigrate, registeredModels...)
			if err != nil {
				zap.S().Errorf("Error migrating, err: %+v", err)
			}
		})
	})
	return err
}

// ResetModels resets registered models and migration state (useful for unit testing).
func ResetModels() {
	registeredModels = nil
	migrateOnce = sync.Once{}
}
