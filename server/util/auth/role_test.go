package auth_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"github.com/stretchr/testify/assert"
)

func TestStrToUserRole(t *testing.T) {
	tests := []struct {
		input   string
		want    auth.Role
		wantErr error
	}{
		{"admin", auth.ROLE_ADMIN, nil},
		{"superadmin", auth.ROLE_ADMIN, nil},
		{"user", auth.ROLE_USER, nil},
		{"guest", "", cerror.ErrUnknownRole},
		{"", "", cerror.ErrUnknownRole},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := auth.StrToRole(tt.input)
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
		assert.NoError(t, auth.ValidateRole(auth.ROLE_USER))

		assert.NoError(t, auth.ValidateRole(auth.ROLE_ADMIN))
	})

	t.Run("Invalid role", func(t *testing.T) {
		assert.Error(t, auth.ValidateRole(auth.Role("invalid_role")))
	})
}

func TestUserRole_IsValid(t *testing.T) {
	t.Run("Valid roles", func(t *testing.T) {
		assert.True(t, auth.ROLE_USER.IsValid())
		assert.True(t, auth.ROLE_ADMIN.IsValid())
	})

	t.Run("Invalid roles", func(t *testing.T) {
		assert.False(t, auth.Role("invalid_role").IsValid())
		assert.False(t, auth.Role("").IsValid())
	})
}
