package testutil

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/db"
	"gorm.io/gorm"
)

// NewSqliteTestDb creates a fast, isolated in-memory SQLite database instance for testing.
// It uses pure-Go glebarez/sqlite (no CGO requirement) and auto-migrates provided model schemas.
func NewSqliteTestDb(t *testing.T, models ...any) *gorm.DB {
	gormDb := db.NewSqliteMemoryDb(models...)

	if len(models) > 0 {
		t.Cleanup(func() {
			_ = gormDb.Migrator().DropTable(models...)
		})
	}

	return gormDb
}
