package authutils_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSetRefreshTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Sets correct cookie attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/auth/login", nil)

		authutils.SetRefreshTokenCookie(c, "sample-refresh-token")

		cookie := findCookie(w.Result().Cookies(), authutils.RefreshTokenCookieName)
		assert.NotNil(t, cookie)
		assert.Equal(t, "sample-refresh-token", cookie.Value)
		assert.Equal(t, authutils.RefreshTokenCookiePath, cookie.Path)
		assert.Equal(t, authutils.RefreshTokenCookieMaxAge, cookie.MaxAge)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.False(t, cookie.Secure)
	})

	t.Run("Cookie secure flag variations", func(t *testing.T) {
		tests := []struct {
			name       string
			setupReq   func() *http.Request
			build      string
			wantSecure bool
		}{
			{
				name: "TLS connection",
				setupReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
					req.TLS = &tls.ConnectionState{}
					return req
				},
				wantSecure: true,
			},
			{
				name: "X-Forwarded-Proto is https",
				setupReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
					req.Header.Set("X-Forwarded-Proto", "https")
					return req
				},
				wantSecure: true,
			},
			{
				name: "Localhost with port",
				setupReq: func() *http.Request {
					return httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
				},
				wantSecure: false,
			},
			{
				name: "127.0.0.1 with port",
				setupReq: func() *http.Request {
					return httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
				},
				wantSecure: false,
			},
			{
				name: "[::1] with port",
				setupReq: func() *http.Request {
					return httptest.NewRequest(http.MethodGet, "http://[::1]:8080", nil)
				},
				wantSecure: false,
			},
			{
				name: "Remote host in prod",
				setupReq: func() *http.Request {
					return httptest.NewRequest(http.MethodGet, "http://example.com:8080", nil)
				},
				build:      app.BuildProd,
				wantSecure: true,
			},
			{
				name: "Remote host not in prod",
				setupReq: func() *http.Request {
					return httptest.NewRequest(http.MethodGet, "http://example.com:8080", nil)
				},
				build:      app.BuildDev,
				wantSecure: false,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				oldBuild := app.Build
				defer func() { app.Build = oldBuild }()
				if tc.build != "" {
					app.Build = tc.build
				}

				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = tc.setupReq()

				authutils.SetRefreshTokenCookie(c, "token")

				cookie := findCookie(w.Result().Cookies(), authutils.RefreshTokenCookieName)
				assert.NotNil(t, cookie)
				assert.Equal(t, tc.wantSecure, cookie.Secure)
			})
		}
	})
}

func TestClearRefreshTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/auth/logout", nil)

	authutils.ClearRefreshTokenCookie(c)

	cookie := findCookie(w.Result().Cookies(), authutils.RefreshTokenCookieName)
	assert.NotNil(t, cookie)
	assert.Empty(t, cookie.Value)
	assert.Equal(t, -1, cookie.MaxAge)
	assert.Equal(t, authutils.RefreshTokenCookiePath, cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}
