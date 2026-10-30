package user_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUserDto_ToModel(t *testing.T) {
	tests := []struct {
		name    string
		dto     user.NewUserDto
		want    *user.User
		wantErr error
	}{
		{
			name: "Valid DTO to Model - No UUID in DTO",
			dto: user.NewUserDto{
				Username: "testuser",
				Email:    "test.user@example.com",
				Password: "password123",
				Role:     string(authutils.ROLE_USER),
			},
			want: &user.User{
				Username: "testuser",
				Email:    "test.user@example.com",
				Role:     authutils.ROLE_USER,
			},
			wantErr: nil,
		},
		{
			name: "Valid DTO to Model - Admin Role",
			dto: user.NewUserDto{
				Username: "anotheruser",
				Email:    "another.user@example.com",
				Password: "securepass",
				Role:     string(authutils.ROLE_ADMIN),
			},
			want: &user.User{
				Username: "anotheruser",
				Email:    "another.user@example.com",
				Role:     authutils.ROLE_ADMIN,
			},
			wantErr: nil,
		},
		{
			name: "Invalid Role",
			dto: user.NewUserDto{
				Username: "badrole",
				Role:     "nonexistent_role",
				Email:    "bad.role@example.com",
				Password: "password",
			},
			want:    nil,
			wantErr: cerror.ErrUnknownRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dto.ToModel()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.NotEqual(t, uuid.Nil, got.Uuid)
				assert.Equal(t, tt.want.Username, got.Username)
				assert.Equal(t, tt.want.Email, got.Email)
				assert.Equal(t, tt.want.Role, got.Role)
			}
		})
	}
}

func TestUserDto_ToModel(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name    string
		dto     user.UserDto
		want    *user.User
		wantErr error
	}{
		{
			name: "Valid DTO to Model",
			dto: user.UserDto{
				Uuid:     validUUID.String(),
				Username: "johndoe",
				Email:    "john.doe@example.com",
				Role:     "user",
			},
			want: &user.User{
				Uuid:     validUUID,
				Username: "johndoe",
				Email:    "john.doe@example.com",
				Role:     authutils.ROLE_USER,
			},
			wantErr: nil,
		},
		{
			name: "Invalid UUID",
			dto: user.UserDto{
				Uuid:     "not-a-uuid",
				Username: "janedoe",
				Role:     "user",
			},
			want:    nil,
			wantErr: cerror.ErrBadUuid,
		},
		{
			name: "Invalid Role",
			dto: user.UserDto{
				Uuid:     validUUID.String(),
				Username: "jackdaniels",
				Role:     "invalid_role",
			},
			want:    nil,
			wantErr: cerror.ErrUnknownRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dto.ToModel()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Uuid, got.Uuid)
				assert.Equal(t, tt.want.Username, got.Username)
				assert.Equal(t, tt.want.Email, got.Email)
				assert.Equal(t, tt.want.Role, got.Role)
			}
		})
	}
}

func TestUserDto_FromModel(t *testing.T) {
	userUUID := uuid.New()
	userModel := &user.User{
		Uuid:     userUUID,
		Username: "alicesmith",
		Email:    "alice.smith@example.com",
		Role:     authutils.ROLE_USER,
	}

	expectedDto := user.UserDto{
		Uuid:     userUUID.String(),
		Username: "alicesmith",
		Email:    "alice.smith@example.com",
		Role:     string(authutils.ROLE_USER),
	}

	var gotDto user.UserDto
	gotDto = gotDto.FromModel(userModel)

	assert.Equal(t, expectedDto, gotDto)
}
