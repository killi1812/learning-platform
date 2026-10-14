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
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockAuthService struct {
	loginFunc         func(email, password, userAgent, ip, requestedSessionId string) (*AuthResult, error)
	registerFunc      func(username, email, password, userAgent, ip string) (*AuthResult, error)
	refreshTokensFunc func(refreshToken string) (*AuthResult, error)
	logoutCurrentFunc func(userUuid uuid.UUID, sessionUuid uuid.UUID) error
	logoutAllFunc     func(userUuid uuid.UUID) error
	listSessionsFunc  func(userUuid, currentSessionUuid uuid.UUID) ([]SessionDto, error)
	revokeSessionFunc func(userUuid, targetSessionUuid uuid.UUID) error
}

func (m *mockAuthService) Login(email, password, userAgent, ip, requestedSessionId string) (*AuthResult, error) {
	if m.loginFunc != nil {
		return m.loginFunc(email, password, userAgent, ip, requestedSessionId)
	}
	return &AuthResult{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		SessionId:    "mock-session-id",
	}, nil
}

func (m *mockAuthService) Register(username, email, password, userAgent, ip string) (*AuthResult, error) {
	if m.registerFunc != nil {
		return m.registerFunc(username, email, password, userAgent, ip)
	}
	return &AuthResult{
		AccessToken:  "mock-registered-access-token",
		RefreshToken: "mock-registered-refresh-token",
		SessionId:    "mock-registered-session-id",
	}, nil
}

func (m *mockAuthService) RefreshTokens(refreshToken string) (*AuthResult, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(refreshToken)
	}
	return &AuthResult{
		AccessToken:  "mock-refreshed-token",
		RefreshToken: "mock-new-refresh-token",
		SessionId:    "mock-refreshed-session-id",
	}, nil
}

func (m *mockAuthService) LogoutCurrent(userUuid uuid.UUID, sessionUuid uuid.UUID) error {
	if m.logoutCurrentFunc != nil {
		return m.logoutCurrentFunc(userUuid, sessionUuid)
	}
	return nil
}

func (m *mockAuthService) LogoutAll(userUuid uuid.UUID) error {
	if m.logoutAllFunc != nil {
		return m.logoutAllFunc(userUuid)
	}
	return nil
}

func (m *mockAuthService) ListSessions(userUuid, currentSessionUuid uuid.UUID) ([]SessionDto, error) {
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

func (m *mockAuthService) RevokeSession(userUuid, targetSessionUuid uuid.UUID) error {
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
			loginFunc: func(email, pass, ua, ip, sessId string) (*AuthResult, error) {
				assert.Equal(t, "user1@example.com", email)
				assert.Equal(t, "pass1", pass)
				assert.Equal(t, "sess-123", sessId)
				return &AuthResult{
					AccessToken:  "jwt-token-abc",
					RefreshToken: "jwt-refresh-xyz",
					SessionId:    "sess-123",
				}, nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		payload := LoginDto{Email: "user1@example.com", Password: "pass1", SessionId: "sess-123"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokenResp TokenDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokenResp))
		assert.Equal(t, "jwt-token-abc", tokenResp.AccessToken)

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.Equal(t, "jwt-refresh-xyz", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.Equal(t, RefreshTokenCookiePath, refreshCookie.Path)
	})

	t.Run("Invalid credentials returns 401 with JSON error", func(t *testing.T) {
		mock := &mockAuthService{
			loginFunc: func(email, pass, ua, ip, sessId string) (*AuthResult, error) {
				return nil, errors.New("invalid credentials")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		payload := LoginDto{Email: "wrong@example.com", Password: "wrong"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid email or password")
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
	t.Run("Valid refresh with cookie returns 200 with new tokens and rotated cookie", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (*AuthResult, error) {
				assert.Equal(t, "cookie-refresh-token", token)
				return &AuthResult{
					AccessToken:  "new-refreshed-token",
					RefreshToken: "new-rotated-refresh-token",
					SessionId:    "sess-cookie",
				}, nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  RefreshTokenCookieName,
			Value: "cookie-refresh-token",
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokenResp TokenDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokenResp))
		assert.Equal(t, "new-refreshed-token", tokenResp.AccessToken)

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.Equal(t, "new-rotated-refresh-token", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.Equal(t, RefreshTokenCookiePath, refreshCookie.Path)
	})

	t.Run("Valid refresh with Authorization header returns 200 with new tokens", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (*AuthResult, error) {
				return &AuthResult{
					AccessToken:  "new-refreshed-token",
					RefreshToken: "new-refresh-token",
					SessionId:    "sess-1",
				}, nil
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

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.Equal(t, "new-refresh-token", refreshCookie.Value)
	})

	t.Run("Missing refresh token returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Refresh service error returns 401 and clears cookie", func(t *testing.T) {
		mock := &mockAuthService{
			refreshTokensFunc: func(token string) (*AuthResult, error) {
				return nil, errors.New("refresh failed")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer invalid-refresh-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.True(t, refreshCookie.MaxAge < 0 || refreshCookie.Value == "")
	})
}

func TestAuthCtn_DualLogout(t *testing.T) {
	userUuid := uuid.New()
	sessionUuid := uuid.New()

	t.Run("Valid single session logout returns 204 and clears cookie", func(t *testing.T) {
		mock := &mockAuthService{
			logoutCurrentFunc: func(uId uuid.UUID, sId uuid.UUID) error {
				assert.Equal(t, userUuid, uId)
				assert.Equal(t, sessionUuid, sId)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.True(t, refreshCookie.MaxAge < 0 || refreshCookie.Value == "")
	})

	t.Run("Valid logout-all returns 204 and clears cookie", func(t *testing.T) {
		mock := &mockAuthService{
			logoutAllFunc: func(uId uuid.UUID) error {
				assert.Equal(t, userUuid, uId)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.True(t, refreshCookie.MaxAge < 0 || refreshCookie.Value == "")
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
			logoutCurrentFunc: func(id uuid.UUID, sId uuid.UUID) error {
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
			listSessionsFunc: func(uId, sId uuid.UUID) ([]SessionDto, error) {
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

		req, _ := http.NewRequest(http.MethodGet, "/api/auth/session", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var sessions []SessionDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessions))
		assert.Len(t, sessions, 1)
		assert.True(t, sessions[0].IsCurrent)
	})

	t.Run("Revoke specific session returns 204", func(t *testing.T) {
		targetSession := uuid.New()
		mock := &mockAuthService{
			revokeSessionFunc: func(uId, target uuid.UUID) error {
				assert.Equal(t, userUuid, uId)
				assert.Equal(t, targetSession, target)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodDelete, "/api/auth/session/"+targetSession.String(), nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Revoke with invalid session UUID returns 400", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodDelete, "/api/auth/session/not-a-uuid", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid, sessionUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthCtn_Register(t *testing.T) {
	t.Run("Valid register returns 201 with tokens and cookie", func(t *testing.T) {
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

		var refreshCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == RefreshTokenCookieName {
				refreshCookie = c
				break
			}
		}
		assert.NotNil(t, refreshCookie)
		assert.Equal(t, "mock-registered-refresh-token", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.Equal(t, RefreshTokenCookiePath, refreshCookie.Path)
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
			registerFunc: func(username, email, password, ua, ip string) (*AuthResult, error) {
				return nil, errors.New("username or email already exists")
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
