package auth_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseToken(t *testing.T) {
	app.AccessKey = "test-jwt-key"
	app.RefreshKey = "test-refresh-key"

	now := time.Now()
	accessTokenDuration := 5 * time.Minute
	validClaims := auth.Claims{
		Username: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
			ID:        "test-uuid",
		},
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := validToken.SignedString([]byte(app.AccessKey))
	validAuthHeader := "Bearer " + validTokenString

	invalidSignatureToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	invalidSignatureTokenString, _ := invalidSignatureToken.SignedString([]byte("wrong-key"))
	invalidSignatureAuthHeader := "Bearer " + invalidSignatureTokenString

	expiredClaims := auth.Claims{
		Username: "expired@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			ID:        "expired-uuid",
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte(app.AccessKey))
	expiredAuthHeader := "Bearer " + expiredTokenString

	tests := []struct {
		name       string
		authHeader string
		wantToken  bool
		wantClaims *auth.Claims
		wantErr    error
	}{
		{
			name:       "Valid token",
			authHeader: validAuthHeader,
			wantToken:  true,
			wantClaims: &validClaims,
			wantErr:    nil,
		},
		{
			name:       "Invalid token format - missing Bearer",
			authHeader: validTokenString,
			wantToken:  false,
			wantClaims: nil,
			wantErr:    cerror.ErrInvalidTokenFormat,
		},
		{
			name:       "Invalid token format - too short",
			authHeader: "Bearer",
			wantToken:  false,
			wantClaims: nil,
			wantErr:    cerror.ErrInvalidTokenFormat,
		},
		{
			name:       "Invalid signature",
			authHeader: invalidSignatureAuthHeader,
			wantToken:  false,
			wantClaims: nil,
			wantErr:    jwt.ErrSignatureInvalid,
		},
		{
			name:       "Expired token",
			authHeader: expiredAuthHeader,
			wantToken:  false,
			wantClaims: nil,
			wantErr:    jwt.ErrTokenExpired,
		},
		{
			name:       "Malformed token",
			authHeader: "Bearer malformed.token.string",
			wantToken:  false,
			wantClaims: nil,
			wantErr:    jwt.ErrTokenMalformed,
		},
	}
	origBuild := app.Build
	app.Build = app.BuildProd
	defer func() { app.Build = origBuild }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotClaims, gotErr := auth.ParseToken(tt.authHeader)

			if gotErr != nil {
				if tt.wantErr == nil || !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("ParseToken() error = %v, wantErr %v", gotErr, tt.wantErr)
				}
				return
			}
			if tt.wantErr != nil {
				t.Errorf("ParseToken() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if (gotToken != nil) != tt.wantToken {
				t.Errorf("ParseToken() gotToken = %v, wantToken %v", gotToken != nil, tt.wantToken)
			}

			if tt.wantClaims != nil {
				if gotClaims == nil {
					t.Errorf("ParseToken() gotClaims = nil, want non-nil")
					return
				}
				if !reflect.DeepEqual(gotClaims, tt.wantClaims) {
					t.Errorf("ParseToken() gotClaims = %v, wantClaims %v", gotClaims, tt.wantClaims)
				}
			} else if gotClaims != nil {
				t.Errorf("ParseToken() gotClaims = %v, want nil", gotClaims)
			}
		})
	}

	t.Run("Dev mode accepts token without Bearer prefix", func(t *testing.T) {
		app.Build = app.BuildDev
		gotToken, gotClaims, gotErr := auth.ParseToken(validTokenString)
		if gotErr != nil {
			t.Fatalf("ParseToken() unexpected error in dev mode: %v", gotErr)
		}
		if gotToken == nil || gotClaims == nil {
			t.Fatalf("ParseToken() returned nil token or claims in dev mode")
		}
		if gotClaims.Username != validClaims.Username {
			t.Errorf("ParseToken() username = %s, want %s", gotClaims.Username, validClaims.Username)
		}
	})
}

func TestGenerateTokens(t *testing.T) {
	app.AccessKey = "test-jwt-key"
	app.RefreshKey = "test-refresh-key"

	userUuid := uuid.New()
	sessionUuid := uuid.New()

	access, refresh, err := auth.GenerateTokens("user@example.com", "user", auth.ROLE_USER, userUuid, sessionUuid)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)

	// Validate access token claims
	_, claims, err := auth.ParseToken("Bearer " + access)
	assert.NoError(t, err)
	assert.Equal(t, userUuid.String(), claims.Subject)
	assert.Equal(t, userUuid.String(), claims.ID)
	assert.Equal(t, sessionUuid, claims.SessionUuid)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "user", claims.Username)
	assert.Equal(t, auth.ROLE_USER, claims.Role)
}

