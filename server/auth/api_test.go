package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockAuthService struct {
	loginFunc         func(identifier, password, userAgent, ip string) (string, string, error)
	registerFunc      func(username, email, password, userAgent, ip string) (string, string, error)
	refreshTokensFunc func(refreshToken string) (string, string, error)
	logoutCurrentFunc func(userUuid string, sessionUuid uuid.UUID) error
	logoutAllFunc     func(userUuid string) error
	listSessionsFunc  func(userUuid string, currentSessionUuid uuid.UUID) ([]SessionDto, error)
	revokeSessionFunc func(userUuid string, targetSessionUuid uuid.UUID) error
}

func (m *mockAuthService) Login(identifier, password, userAgent, ip string) (string, string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(identifier, password, userAgent, ip)
	}
	return "mock-access-token", "mock-refresh-token", nil
}

func (m *mockAuthService) Register(username, email, password, userAgent, ip string) (string, string, error) {
	if m.registerFunc != nil {
		return m.registerFunc(username, email, password, userAgent, ip)
	}
	return "mock-registered-access-token", "mock-registered-refresh-token", nil
}

func (m *mockAuthService) RefreshTokens(refreshToken string) (string, string, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(refreshToken)
	}
	return "mock-refreshed-token", "mock-new-refresh-token", nil
}

func (m *mockAuthService) LogoutCurrent(userUuid string, sessionUuid uuid.UUID) error {
	if m.logoutCurrentFunc != nil {
		return m.logoutCurrentFunc(userUuid, sessionUuid)
	}
	return nil
}

func (m *mockAuthService) LogoutAll(userUuid string) error {
	if m.logoutAllFunc != nil {
		return m.logoutAllFunc(userUuid)
	}
	return nil
}

func (m *mockAuthService) ListSessions(userUuid string, currentSessionUuid uuid.UUID) ([]SessionDto, error) {
	if m.listSessionsFunc != nil {
		return m.listSessionsFunc(userUuid, currentSessionUuid)
	}
	return []SessionDto{
		{
			Id:         currentSessionUuid.String(),
			DeviceName: "Chrome on Linux",
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
			IsCurrent:  true,
		},
	}, nil
}

func (m *mockAuthService) RevokeSession(userUuid string, targetSessionUuid uuid.UUID) error {
	if m.revokeSessionFunc != nil {
		return m.revokeSessionFunc(userUuid, targetSessionUuid)
	}
	return nil
}

func setupAuthTestRouter(service IAuthService) (*gin.Engine, *AuthCtn) {
	gin.SetMode(gin.TestMode)
	app.AccessKey = "test-auth-secret-access-key-12345"
	app.RefreshKey = "test-auth-secret-refresh-key-12345"

	logger := zap.NewNop().Sugar()
	ctn := &AuthCtn{
		auth:   service,
		logger: logger,
	}

	r := gin.New()
	api := r.Group("/api")
	ctn.RegisterEndpoints(api)
	return r, ctn
}

func generateValidToken(t *testing.T, userUuid, sessionUuid uuid.UUID) string {
	token, _, err := authUtil.GenerateTokens("auth@test.com", "authuser", "user", userUuid, sessionUuid)
	assert.NoError(t, err)
	return "Bearer " + token
}

func TestAuthCtn_Login(t *testing.T) {
	t.Run("Valid login returns 200 with TokenDto", func(t *testing.T) {
		mock := &mockAuthService{
			loginFunc: func(id, pass, ua, ip string) (string, string, error) {
				assert.Equal(t, "user1", id)
				assert.Equal(t, "pass1", pass)
				return "jwt-token-abc", "jwt-refresh-xyz", nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		payload := LoginDto{Username: "user1", Password: "pass1"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokenResp TokenDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokenResp))
		assert.Equal(t, "jwt-token-abc", tokenResp.AccessToken)
		assert.Equal(t, "jwt-refresh-xyz", tokenResp.RefreshToken)
	})

	t.Run("Invalid credentials returns 401 with JSON error", func(t *testing.T) {
		mock := &mockAuthService{
			loginFunc: func(id, pass, ua, ip string) (string, string, error) {
				return "", "", errors.New("invalid credentials")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		payload := LoginDto{Username: "wrong", Password: "wrong"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid username or password")
	})

	t.Run("Malformed JSON returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("bad json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthCtn_RefreshToken(t *testing.T) {
	t.Run("Valid refresh with Authorization header returns 200 with new tokens", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (string, string, error) {
				return "new-refreshed-token", "new-refresh-token", nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer valid-refresh-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokenResp TokenDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokenResp))
		assert.Equal(t, "new-refreshed-token", tokenResp.AccessToken)
		assert.Equal(t, "new-refresh-token", tokenResp.RefreshToken)
	})

	t.Run("Valid refresh with JSON body returns 200 with new tokens", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (string, string, error) {
				assert.Equal(t, "body-refresh-token", token)
				return "new-access", "new-refresh", nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		body, _ := json.Marshal(RefreshDto{RefreshToken: "body-refresh-token"})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokenResp TokenDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokenResp))
		assert.Equal(t, "new-access", tokenResp.AccessToken)
		assert.Equal(t, "new-refresh", tokenResp.RefreshToken)
	})

	t.Run("Missing refresh token returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Refresh service error returns 401", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (string, string, error) {
				return "", "", errors.New("refresh failed")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer invalid-refresh-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthCtn_DualLogout(t *testing.T) {
	userUuid := uuid.New()
	sessionUuid := uuid.New()

	t.Run("Valid single session logout returns 200", func(t *testing.T) {
		mock := &mockAuthService{
			logoutCurrentFunc: func(uId string, sId uuid.UUID) error {
				assert.Equal(t, userUuid.String(), uId)
				assert.Equal(t, sessionUuid, sId)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Logged out successfully")
	})

	t.Run("Valid logout-all returns 200", func(t *testing.T) {
		mock := &mockAuthService{
			logoutAllFunc: func(uId string) error {
				assert.Equal(t, userUuid.String(), uId)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "All sessions logged out successfully")
	})

	t.Run("Logout without token returns 401", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Logout service error returns 500", func(t *testing.T) {
		mock := &mockAuthService{
			logoutCurrentFunc: func(id string, sId uuid.UUID) error {
				return errors.New("logout error")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthCtn_Sessions(t *testing.T) {
	userUuid := uuid.New()
	sessionUuid := uuid.New()

	t.Run("List sessions returns 200 with session list", func(t *testing.T) {
		mock := &mockAuthService{
			listSessionsFunc: func(uId string, sId uuid.UUID) ([]SessionDto, error) {
				return []SessionDto{
					{
						Id:         sId.String(),
						DeviceName: "Chrome on Linux",
						CreatedAt:  time.Now(),
						ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
						IsCurrent:  true,
					},
				}, nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var sessions []SessionDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessions))
		assert.Len(t, sessions, 1)
		assert.True(t, sessions[0].IsCurrent)
	})

	t.Run("Revoke specific session returns 200", func(t *testing.T) {
		targetSession := uuid.New()
		mock := &mockAuthService{
			revokeSessionFunc: func(uId string, target uuid.UUID) error {
				assert.Equal(t, targetSession, target)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodDelete, "/api/auth/sessions/"+targetSession.String(), nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Session revoked successfully")
	})

	t.Run("Revoke with invalid session UUID returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodDelete, "/api/auth/sessions/not-a-uuid", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthCtn_Register(t *testing.T) {
	t.Run("Valid register returns 201 with tokens", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		body, _ := json.Marshal(RegisterDto{
			Username: "newuser",
			Email:    "newuser@example.com",
			Password: "securePassword123",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp TokenDto
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "mock-registered-access-token", resp.AccessToken)
		assert.Equal(t, "mock-registered-refresh-token", resp.RefreshToken)
	})

	t.Run("Invalid register payload returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Duplicate user registration returns 400 with error message", func(t *testing.T) {
		mock := &mockAuthService{
			registerFunc: func(username, email, password, ua, ip string) (string, string, error) {
				return "", "", errors.New("username or email already exists")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		body, _ := json.Marshal(RegisterDto{
			Username: "existing",
			Email:    "existing@example.com",
			Password: "securePassword123",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Username or email already exists")
	})
}

func TestNewAuthCtn(t *testing.T) {
	ctn := NewAuthCtn(&mockAuthService{}, zap.NewNop().Sugar())
	assert.NotNil(t, ctn)
}
