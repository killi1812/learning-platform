package user_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_Update(t *testing.T) {
	oldTime := time.Now().Add(-1 * time.Hour)
	u := &user.User{
		Uuid:         uuid.New(),
		Username:     "oldname",
		Email:        "old@example.com",
		Role:         authutils.ROLE_USER,
		PasswordHash: "secret",
	}
	u.UpdatedAt = oldTime

	newData := &user.User{
		Username: "newname",
		Email:    "new@example.com",
		Role:     authutils.ROLE_ADMIN,
	}

	res := u.Update(newData)
	assert.Equal(t, "newname", res.Username)
	assert.Equal(t, "new@example.com", res.Email)
	assert.Equal(t, authutils.ROLE_ADMIN, res.Role)
	assert.Equal(t, "secret", res.PasswordHash) // preserved
	assert.True(t, res.UpdatedAt.After(oldTime))
}
