package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewSqliteDb initializes and returns an SQLite database connection using pure-Go glebarez/sqlite (no CGO required).
func NewSqliteDb(dsn string, isDev bool, models ...any) *gorm.DB {
	if dsn == "" {
		dsn = "learning_platform.db"
	}

	var gormLogLevel logger.LogLevel
	if isDev {
		gormLogLevel = logger.Info
	} else {
		gormLogLevel = logger.Warn
	}

	gormLogger := newGormZapLogger().LogMode(gormLogLevel)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		zap.S().Panicf("failed to open sqlite database %s: %+v", dsn, err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			zap.S().Panicf("failed to auto-migrate sqlite database %s: %+v", dsn, err)
		}
	}

	return db
}

// NewSqliteMemoryDb initializes and returns a fast in-memory SQLite database connection using pure-Go glebarez/sqlite.
func NewSqliteMemoryDb(models ...any) *gorm.DB {
	memDsn := fmt.Sprintf("file:mem_%s?mode=memory&cache=shared", uuid.New().String()[:8])
	db, err := gorm.Open(sqlite.Open(memDsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		zap.S().Panicf("failed to open in-memory sqlite database: %+v", err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			zap.S().Panicf("failed to auto-migrate in-memory sqlite database: %+v", err)
		}
	}

	return db
}
