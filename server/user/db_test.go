package user

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestUserCrudService_CRUD(t *testing.T) {
	db := testutil.NewSqliteTestDb(t, &User{})
	logger := zap.NewNop().Sugar()
	svc := &UserCrudService{db: db, logger: logger}

	var createdUser *User

	t.Run("Create user hashes password and sets fields", func(t *testing.T) {
		u := &User{
			Username: "sigmar",
			Email:    "sigmar@empire.com",
			Role:     authutils.ROLE_ADMIN,
		}
		res, err := svc.Create(u, "hammerofsigmar")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEqual(t, uuid.Nil, res.Uuid)
		assert.NotEmpty(t, res.PasswordHash)
		assert.False(t, res.CreatedAt.IsZero())
		createdUser = res
	})

	t.Run("Create duplicate username returns error", func(t *testing.T) {
		uDup := &User{
			Username: "sigmar",
			Email:    "another@empire.com",
			Role:     authutils.ROLE_USER,
		}
		_, err := svc.Create(uDup, "somepass")
		assert.Error(t, err)
	})

	t.Run("Read existing user", func(t *testing.T) {
		found, err := svc.Read(createdUser.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "sigmar", found.Username)
		assert.Equal(t, "sigmar@empire.com", found.Email)
	})

	t.Run("Read non-existent user returns ErrRecordNotFound", func(t *testing.T) {
		_, err := svc.Read(uuid.New())
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("GetUserByEmail finds existing user", func(t *testing.T) {
		found, err := svc.GetUserByEmail("sigmar@empire.com")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, createdUser.Uuid, found.Uuid)
	})

	t.Run("GetUserByEmail non-existent returns ErrRecordNotFound", func(t *testing.T) {
		_, err := svc.GetUserByEmail("unknown@nowhere.com")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Create second user and ReadAll / GetAllUsers", func(t *testing.T) {
		u2 := &User{
			Username: "karlfranz",
			Email:    "karl@empire.com",
			Role:     authutils.ROLE_USER,
		}
		_, err := svc.Create(u2, "emperorpass")
		assert.NoError(t, err)

		all1, err := svc.ReadAll()
		assert.NoError(t, err)
		assert.Len(t, all1, 2)

		all2, err := svc.GetAllUsers()
		assert.NoError(t, err)
		assert.Len(t, all2, 2)
	})

	t.Run("SearchUsersByName matches close query", func(t *testing.T) {
		matches, err := svc.SearchUsersByName("sigmar")
		assert.NoError(t, err)
		assert.NotEmpty(t, matches)
		assert.Equal(t, "sigmar", matches[0].Username)

		noMatches, err := svc.SearchUsersByName("completelyunrelatedstring12345")
		assert.NoError(t, err)
		assert.Empty(t, noMatches)
	})

	t.Run("Update modifies user", func(t *testing.T) {
		toUpdate := &User{
			Username: "sigmar_unifier",
			Email:    "unifier@empire.com",
			Role:     authutils.ROLE_ADMIN,
		}
		updated, err := svc.Update(createdUser.Uuid, toUpdate)
		assert.NoError(t, err)
		assert.Equal(t, "sigmar_unifier", updated.Username)
		assert.Equal(t, "unifier@empire.com", updated.Email)
	})

	t.Run("Update non-existent user returns ErrRecordNotFound", func(t *testing.T) {
		_, err := svc.Update(uuid.New(), &User{Username: "ghost"})
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Delete removes user", func(t *testing.T) {
		err := svc.Delete(createdUser.Uuid)
		assert.NoError(t, err)

		_, err = svc.Read(createdUser.Uuid)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Delete non-existent user returns ErrRecordNotFound", func(t *testing.T) {
		err := svc.Delete(createdUser.Uuid)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func TestNewUserCrudService(t *testing.T) {
	db := testutil.NewSqliteTestDb(t, &User{})
	svc := NewUserCrudService(db, zap.NewNop().Sugar())
	assert.NotNil(t, svc)
}
