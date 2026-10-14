package user

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

type Session struct {
	SessionUuid  uuid.UUID `json:"sessionUuid" bson:"session_uuid"`
	UserUuid     uuid.UUID `json:"userUuid" bson:"user_uuid"`
	TokenUuid    uuid.UUID `json:"tokenUuid" bson:"token_uuid"`
	RefreshToken string    `json:"refreshToken" bson:"refresh_token"`
	DeviceName   string    `json:"deviceName" bson:"device_name"`
	IpAddress    string    `json:"ipAddress,omitempty" bson:"ip_address,omitempty"`
	CreatedAt    time.Time `json:"createdAt" bson:"created_at"`
	ExpiresAt    time.Time `json:"expiresAt" bson:"expires_at"`
}

type User struct {
	gorm.Model
	Uuid         uuid.UUID      `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	Username     string         `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	Email        string         `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string         `json:"passwordHash" gorm:"type:varchar(255);not null"`
	Role         authutils.Role `json:"role" gorm:"type:varchar(32);not null;default:'user'"`
}

func (u *User) Update(user *User) *User {
	u.Username = user.Username
	u.Email = user.Email
	u.Role = user.Role
	u.UpdatedAt = time.Now()

	return u
}
