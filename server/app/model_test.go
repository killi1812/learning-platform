package app

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type dummyAppModel struct {
	ID    uint   `gorm:"primaryKey"`
	Title string `gorm:"type:varchar(100)"`
}

func TestModelRegistrationAndMigration(t *testing.T) {
	origBuild := Build
	origRunMigrate := RunMigrate
	defer func() {
		Build = origBuild
		RunMigrate = origRunMigrate
	}()

	logger := zap.NewNop()
	zap.ReplaceGlobals(logger)

	t.Run("RegisterModel registers models and Migrate applies schema", func(t *testing.T) {
		Test()
		Build = BuildDev
		db := testutil.NewSqliteTestDb(t)
		Provide(func() *gorm.DB { return db })

		RegisterModel(&dummyAppModel{})
		assert.Equal(t, 1, len(registeredModels))

		err := Migrate()
		assert.NoError(t, err)
		assert.True(t, db.Migrator().HasTable(&dummyAppModel{}))
	})

	t.Run("Migrate runs once even if called multiple times", func(t *testing.T) {
		Test()
		Build = BuildDev
		db := testutil.NewSqliteTestDb(t)
		Provide(func() *gorm.DB { return db })

		RegisterModel(&dummyAppModel{})

		err := Migrate()
		assert.NoError(t, err)

		// Second call should not error
		err = Migrate()
		assert.NoError(t, err)
	})

	t.Run("ResetModels clears registered models and migration state", func(t *testing.T) {
		ResetModels()
		RegisterModel(&dummyAppModel{})
		assert.Equal(t, 1, len(registeredModels))

		ResetModels()
		assert.Equal(t, 0, len(registeredModels))
	})

	t.Run("Migrate returns error when database is closed", func(t *testing.T) {
		Test()
		Build = BuildDev
		closedDb := testutil.NewSqliteTestDb(t)
		sqlDB, err := closedDb.DB()
		assert.NoError(t, err)
		_ = sqlDB.Close()

		Provide(func() *gorm.DB { return closedDb })
		RegisterModel(&dummyAppModel{})

		err = Migrate()
		assert.Error(t, err)
	})
}
