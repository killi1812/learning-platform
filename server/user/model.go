package user

import (
	"errors"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"

	"github.com/google/uuid"
)

type UserRole = auth.UserRole

const (
	ROLE_USER  = auth.ROLE_USER
	ROLE_ADMIN = auth.ROLE_ADMIN
)

var (
	StrToUserRole     = auth.StrToUserRole
	_VALID_USER_ROLES = auth.ValidUserRoles
)

type Session struct {
	UserUuid     uuid.UUID `json:"userUuid" bson:"user_uuid"`
	RefreshToken string    `json:"refreshToken" bson:"refresh_token"`
	CreatedAt    time.Time `json:"createdAt,omitempty" bson:"created_at,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty" bson:"expires_at,omitempty"`
}

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Uuid         uuid.UUID `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	Username     string    `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	Email        string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `json:"passwordHash" gorm:"type:varchar(255);not null"`
	Role         UserRole  `json:"role" gorm:"type:varchar(32);not null;default:'user'"`
	CreatedAt    time.Time `json:"createdAt" gorm:"not null"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"not null"`
}

func (u *User) ValidateRole() error {
	if _, ok := _VALID_USER_ROLES[u.Role]; !ok {
		return errors.New("invalid user role")
	}
	return nil
}

func (u *User) Update(user *User) *User {
	u.Username = user.Username
	u.Email = user.Email
	u.Role = user.Role
	u.UpdatedAt = time.Now()

	return u
}
