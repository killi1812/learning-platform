package auth

import "time"

type LoginDto struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email,omitempty"`
	Password  string `json:"password" binding:"required"`
	SessionId string `json:"sessionId,omitempty"`
}

type TokenDto struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	SessionId    string `json:"sessionId,omitempty"`
}

type RefreshDto struct {
	RefreshToken string `json:"refreshToken"`
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
