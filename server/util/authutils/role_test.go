package authutils_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"github.com/stretchr/testify/assert"
)

func TestStrToUserRole(t *testing.T) {
	tests := []struct {
		input   string
		want    authutils.Role
		wantErr error
	}{
		{"admin", authutils.ROLE_ADMIN, nil},
		{"superadmin", authutils.ROLE_ADMIN, nil},
		{"user", authutils.ROLE_USER, nil},
		{"guest", "", cerror.ErrUnknownRole},
		{"", "", cerror.ErrUnknownRole},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := authutils.StrToRole(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUser_ValidateRole(t *testing.T) {
	t.Run("Valid roles", func(t *testing.T) {
		assert.NoError(t, authutils.ValidateRole(authutils.ROLE_USER))

		assert.NoError(t, authutils.ValidateRole(authutils.ROLE_ADMIN))
	})

	t.Run("Invalid role", func(t *testing.T) {
		assert.Error(t, authutils.ValidateRole(authutils.Role("invalid_role")))
	})
}

func TestUserRole_IsValid(t *testing.T) {
	t.Run("Valid roles", func(t *testing.T) {
		assert.True(t, authutils.ROLE_USER.IsValid())
		assert.True(t, authutils.ROLE_ADMIN.IsValid())
	})

	t.Run("Invalid roles", func(t *testing.T) {
		assert.False(t, authutils.Role("invalid_role").IsValid())
		assert.False(t, authutils.Role("").IsValid())
	})
}
