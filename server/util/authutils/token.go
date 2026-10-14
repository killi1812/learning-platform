package authutils

import (
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Claims struct {
	jwt.RegisteredClaims
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Role        Role      `json:"role"`
	SessionUuid uuid.UUID `json:"sessionUuid"`
	TokenUuid   uuid.UUID `json:"uuid"`
}

const (
	_ACCESS_TOKEN_DURATION  = 5 * time.Minute
	_REFRESH_TOKEN_DURATION = 7 * 24 * time.Hour
)

func ParseToken(authHeader string) (*jwt.Token, *Claims, error) {
	tokenString, err := extractToken(authHeader)
	if err != nil {
		zap.S().Debugf("token: %s", authHeader)
		return nil, nil, err
	}

	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return []byte(app.AccessKey), nil
	})
	if err != nil {
		return nil, nil, err
	}

	return token, &claims, nil
}

func ExtractToken(authHeader string) (string, error) {
	if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		if token == "" {
			return "", cerror.ErrInvalidTokenFormat
		}
		return token, nil
	}

	// In development mode, allow raw JWT tokens without "Bearer " prefix for Swagger UI testing
	if app.Build == app.BuildDev && authHeader != "" && authHeader != "Bearer" {
		return authHeader, nil
	}

	return "", cerror.ErrInvalidTokenFormat
}

func extractToken(authHeader string) (string, error) {
	return ExtractToken(authHeader)
}

// GenerateTokens returns a jwt access token and refresh token or an error.
// Optional sessionUuid preserves the persistent session across token refreshes.
func GenerateTokens(email, username string, role Role, userUuid uuid.UUID, sessionUuid ...uuid.UUID) (string, string, error) {
	if userUuid == uuid.Nil {
		return "", "", cerror.ErrUserIsNil
	}

	activeSessionUuid := uuid.Nil
	if len(sessionUuid) > 0 {
		activeSessionUuid = sessionUuid[0]
	}
	if activeSessionUuid == uuid.Nil {
		activeSessionUuid = uuid.New()
	}

	uuidPair := uuid.New()
	accessTokenClaims := &Claims{
		Email:       email,
		Username:    username,
		Role:        role,
		SessionUuid: activeSessionUuid,
		TokenUuid:   uuidPair,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUuid.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(_ACCESS_TOKEN_DURATION)),
			ID:        userUuid.String(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte(app.AccessKey))
	if err != nil {
		zap.S().Errorf("Failed to generate access token err = %w", err)
		return "", "", err
	}

	refreshTokenClaims := &Claims{
		Username:    username,
		Email:       email,
		Role:        role,
		SessionUuid: activeSessionUuid,
		TokenUuid:   uuidPair,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUuid.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(_REFRESH_TOKEN_DURATION)),
			ID:        userUuid.String(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(app.RefreshKey))
	if err != nil {
		zap.S().Errorf("Failed to generate refresh token err = %w", err)
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}
