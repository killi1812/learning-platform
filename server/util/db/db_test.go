package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type dummyTestModel struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func TestGormZapLogger(t *testing.T) {
	origLogger := zap.L()
	defer zap.ReplaceGlobals(origLogger)

	zap.ReplaceGlobals(zap.NewNop())

	l := newGormZapLogger()
	assert.NotNil(t, l)

	ctx := context.Background()

	t.Run("Log levels", func(t *testing.T) {
		l.Info(ctx, "test info %s", "arg")
		l.Warn(ctx, "test warn %s", "arg")
		l.Error(ctx, "test error %s", "arg")
	})

	t.Run("LogMode changes level", func(t *testing.T) {
		silent := l.LogMode(logger.Silent)
		assert.NotNil(t, silent)
		silent.Info(ctx, "should not output")
		silent.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)
	})

	t.Run("Trace normal query", func(t *testing.T) {
		infoLogger := l.LogMode(logger.Info)
		infoLogger.Trace(ctx, time.Now().Add(-10*time.Millisecond), func() (string, int64) {
			return "SELECT 1", 1
		}, nil)

		infoLogger.Trace(ctx, time.Now().Add(-10*time.Millisecond), func() (string, int64) {
			return "SELECT 1", -1
		}, nil)
	})

	t.Run("Trace slow query", func(t *testing.T) {
		warnLogger := l.LogMode(logger.Warn)
		warnLogger.Trace(ctx, time.Now().Add(-300*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM large_table", 100
		}, nil)

		warnLogger.Trace(ctx, time.Now().Add(-300*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM large_table", -1
		}, nil)
	})

	t.Run("Trace query with error", func(t *testing.T) {
		errLogger := l.LogMode(logger.Error)
		errLogger.Trace(ctx, time.Now().Add(-10*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM nonexistent", -1
		}, errors.New("table not found"))

		errLogger.Trace(ctx, time.Now().Add(-10*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM nonexistent", 0
		}, errors.New("table not found"))
	})
}

func TestHandleMigrations(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())

	t.Run("AutoMigrate in dev mode", func(t *testing.T) {
		db := NewSqliteMemoryDb()

		err := HandleMigrations(db, true, false, &dummyTestModel{})
		assert.NoError(t, err)
		assert.True(t, db.Migrator().HasTable(&dummyTestModel{}))
	})

	t.Run("Migrate flag in prod mode", func(t *testing.T) {
		db := NewSqliteMemoryDb()

		err := HandleMigrations(db, false, true, &dummyTestModel{})
		assert.NoError(t, err)
		assert.True(t, db.Migrator().HasTable(&dummyTestModel{}))
	})

	t.Run("Outdated schema warning in prod mode without migrate flag", func(t *testing.T) {
		db := NewSqliteMemoryDb()

		err := HandleMigrations(db, false, false, &dummyTestModel{})
		assert.NoError(t, err)
		assert.False(t, db.Migrator().HasTable(&dummyTestModel{}))
	})
}

func TestSqliteConstructors(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())

	t.Run("NewSqliteMemoryDb creates in-memory database with models", func(t *testing.T) {
		memDb := NewSqliteMemoryDb(&dummyTestModel{})
		assert.NotNil(t, memDb)
		assert.True(t, memDb.Migrator().HasTable(&dummyTestModel{}))
	})

	t.Run("NewSqliteDb creates file database", func(t *testing.T) {
		tempDsn := t.TempDir() + "/test.db"
		fileDb := NewSqliteDb(tempDsn, true, &dummyTestModel{})
		assert.NotNil(t, fileDb)
		assert.True(t, fileDb.Migrator().HasTable(&dummyTestModel{}))
	})
}
