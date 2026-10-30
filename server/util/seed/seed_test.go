package seed

import (
	"os"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSeed(t *testing.T) {
	db := testutil.NewSqliteTestDb(t, &user.User{})
	app.Test()
	app.Provide(func() *gorm.DB { return db })
	app.Provide(zap.S)
	app.Provide(user.NewUserCrudService)

	origPass := os.Getenv(_PASSWORD_ENV)
	defer os.Setenv(_PASSWORD_ENV, origPass)

	t.Run("Empty password env returns error", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "")
		err := createSuperAdmin()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "env variable is empty")
	})

	t.Run("Short password returns error", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "short")
		err := createSuperAdmin()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})

	t.Run("Valid password creates superadmin and Insert succeeds", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "superpassword123")
		err := createSuperAdmin()
		assert.NoError(t, err)

		// Running again detects existing superadmin and skips
		err = createSuperAdmin()
		assert.NoError(t, err)

		// Insert() wrapper runs without panic
		assert.NotPanics(t, func() {
			Insert()
		})
	})

	t.Run("DB error on GetUserByEmail returns error", func(t *testing.T) {
		closedDb := testutil.NewSqliteTestDb(t, &user.User{})
		sqlDB, err := closedDb.DB()
		assert.NoError(t, err)
		_ = sqlDB.Close()

		app.Test()
		app.Provide(func() *gorm.DB { return closedDb })
		app.Provide(zap.S)
		app.Provide(user.NewUserCrudService)

		err = createSuperAdmin()
		assert.Error(t, err)
	})

	t.Run("Insert handles error gracefully when createSuperAdmin fails", func(t *testing.T) {
		os.Setenv(_PASSWORD_ENV, "")
		assert.NotPanics(t, func() {
			Insert()
		})
	})
}
