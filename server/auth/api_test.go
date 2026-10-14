package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockAuthService struct {
	loginFunc         func(identifier, password string) (string, string, error)
	registerFunc      func(username, email, password string) (string, string, error)
	refreshTokensFunc func(refreshToken string) (string, string, error)
	logoutFunc        func(userUuid string) error
}

func (m *mockAuthService) Login(identifier, password string) (string, string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(identifier, password)
	}
	return "mock-access-token", "mock-refresh-token", nil
}

func (m *mockAuthService) Register(username, email, password string) (string, string, error) {
	if m.registerFunc != nil {
		return m.registerFunc(username, email, password)
	}
	return "mock-registered-access-token", "mock-registered-refresh-token", nil
}

func (m *mockAuthService) RefreshTokens(refreshToken string) (string, string, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(refreshToken)
	}
	return "mock-refreshed-token", "mock-new-refresh-token", nil
}

func (m *mockAuthService) Logout(userUuid string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(userUuid)
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

func generateValidToken(t *testing.T, userUuid uuid.UUID) string {
	token, _, err := authUtil.GenerateTokens("auth@test.com", "authuser", "user", userUuid)
	assert.NoError(t, err)
	return "Bearer " + token
}

func TestAuthCtn_Login(t *testing.T) {
	t.Run("Valid login returns 200 with TokenDto", func(t *testing.T) {
		mock := &mockAuthService{
			loginFunc: func(id, pass string) (string, string, error) {
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

	t.Run("Invalid credentials returns 401", func(t *testing.T) {
		mock := &mockAuthService{
			loginFunc: func(id, pass string) (string, string, error) {
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
	})

	t.Run("Malformed JSON returns without crashing", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("bad json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
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

func TestAuthCtn_Logout(t *testing.T) {
	userUuid := uuid.New()

	t.Run("Valid logout returns 200", func(t *testing.T) {
		mock := &mockAuthService{
			logoutFunc: func(id string) error {
				assert.Equal(t, userUuid.String(), id)
				return nil
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Logout with invalid token returns 401", func(t *testing.T) {
		router, _ := setupAuthTestRouter(&mockAuthService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", "invalid-token-without-bearer")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Direct logout handler without context claims returns 401", func(t *testing.T) {
		_, ctn := setupAuthTestRouter(&mockAuthService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/logout", nil)
		ctn.logout(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Logout service error returns 500", func(t *testing.T) {
		mock := &mockAuthService{
			logoutFunc: func(id string) error {
				return errors.New("logout error")
			},
		}
		router, _ := setupAuthTestRouter(mock)

		req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Authorization", generateValidToken(t, userUuid))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

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
			registerFunc: func(username, email, password string) (string, string, error) {
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
		assert.Contains(t, w.Body.String(), "username or email already exists")
	})
}

func TestNewAuthCtn(t *testing.T) {
	ctn := NewAuthCtn(&mockAuthService{}, zap.NewNop().Sugar())
	assert.NotNil(t, ctn)
}
