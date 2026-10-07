package user

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

type Session struct {
	UserUuid     uuid.UUID `json:"userUuid" bson:"user_uuid"`
	RefreshToken string    `json:"refreshToken" bson:"refresh_token"`
	CreatedAt    time.Time `json:"createdAt,omitempty" bson:"created_at"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty" bson:"expires_at"`
}

type User struct {
	gorm.Model
	Uuid         uuid.UUID `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	Username     string    `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	Email        string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `json:"passwordHash" gorm:"type:varchar(255);not null"`
	Role         auth.Role `json:"role" gorm:"type:varchar(32);not null;default:'user'"`
}

func (u *User) Update(user *User) *User {
	u.Username = user.Username
	u.Email = user.Email
	u.Role = user.Role
	u.UpdatedAt = time.Now()

	return u
}
