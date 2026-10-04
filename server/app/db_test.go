package app

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type dummyModel struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func TestHandleMigrations(t *testing.T) {
	origBuild := Build
	defer func() { Build = origBuild }()

	logger := zap.NewNop()
	zap.ReplaceGlobals(logger)

	t.Run("AutoMigrate in dev mode", func(t *testing.T) {
		Build = BuildDev
		db := testutil.NewSqliteTestDb(t)

		err := HandleMigrations(db, false, &dummyModel{})
		assert.NoError(t, err)
		assert.True(t, db.Migrator().HasTable(&dummyModel{}))
	})

	t.Run("Migrate flag in prod mode", func(t *testing.T) {
		Build = BuildProd
		db := testutil.NewSqliteTestDb(t)

		err := HandleMigrations(db, true, &dummyModel{})
		assert.NoError(t, err)
		assert.True(t, db.Migrator().HasTable(&dummyModel{}))
	})

	t.Run("Outdated schema warning in prod mode without migrate flag", func(t *testing.T) {
		Build = BuildProd
		db := testutil.NewSqliteTestDb(t)

		// Table has not been created yet
		err := HandleMigrations(db, false, &dummyModel{})
		assert.NoError(t, err)
		assert.False(t, db.Migrator().HasTable(&dummyModel{}))
	})
}
