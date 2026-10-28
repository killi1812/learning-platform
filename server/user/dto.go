package user

import (
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type NewUserDto struct {
	Username string `json:"username" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
}

// ToModel create a model from a dto
func (dto NewUserDto) ToModel() (*User, error) {
	role, err := authutils.StrToRole(dto.Role)
	if err != nil {
		zap.S().Errorf("Failed to parse role = %+v, err = %+v", dto.Role, err)
		return nil, cerror.ErrUnknownRole
	}

	return &User{
		Uuid:     uuid.New(),
		Username: dto.Username,
		Email:    dto.Email,
		Role:     role,
	}, nil
}

type UserDto struct {
	Uuid     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func (dto UserDto) ToModel() (*User, error) {
	parsedUuid, err := uuid.Parse(dto.Uuid)
	if err != nil {
		zap.S().Errorf("Failed to parse uuid = %s, err = %+v", dto.Uuid, err)
		return nil, cerror.ErrBadUuid
	}

	role, err := authutils.StrToRole(dto.Role)
	if err != nil {
		zap.S().Errorf("Failed to parse role = %+v, err = %+v", dto.Role, err)
		return nil, cerror.ErrUnknownRole
	}

	return &User{
		Uuid:     parsedUuid,
		Username: dto.Username,
		Email:    dto.Email,
		Role:     role,
	}, nil
}

// FromModel returns a dto from model struct
func (UserDto) FromModel(m *User) UserDto {
	dto := &UserDto{
		Uuid:     m.Uuid.String(),
		Username: m.Username,
		Email:    m.Email,
		Role:     fmt.Sprint(m.Role),
	}
	return *dto
}
