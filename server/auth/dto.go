package auth

import "time"

type LoginDto struct {
	Email     string `json:"email" binding:"required,email" example:"superadmin@example.com"`
	Password  string `json:"password" binding:"required" example:"Pa$$w0rd"`
	SessionId string `json:"sessionId,omitempty" binding:"omitempty"`
}

type TokenDto struct {
	AccessToken string `json:"accessToken"`
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	SessionId    string
}

type RegisterDto struct {
	Username string `json:"username" binding:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type SessionDto struct {
	Id         string    `json:"id"`
	DeviceName string    `json:"deviceName"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IsCurrent  bool      `json:"isCurrent"`
}
