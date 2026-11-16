package db

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDb initializes and returns a PostgreSQL connection via GORM.
func NewPostgresDb(dsn string, isDev, logQuery bool) *gorm.DB {
	if dsn == "" {
		// TODO: Log and return error
	}

	var gormLogLevel logger.LogLevel
	if isDev && logQuery {
		// Info level is debug when it comes to gorm
		gormLogLevel = logger.Info
	} else {
		gormLogLevel = logger.Warn
	}

	gormLogger := newGormZapLogger().LogMode(gormLogLevel)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		zap.S().Panicf("failed to connect to PostgreSQL database: %+v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.S().Panicf("failed to get underlying sql.DB from gorm: %+v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		zap.S().Warnf("PostgreSQL ping warning (server will retry queries on demand): %+v", err)
	} else {
		zap.S().Info("Successfully connected to PostgreSQL")
	}

	return db
}

// NewGormDb is an alias for NewPostgresDb.
func NewGormDb(dsn string, isDev, logQuery bool) *gorm.DB {
	return NewPostgresDb(dsn, isDev, logQuery)
}
