package authutils

import (
	"net"
	"net/http"
	"strings"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"github.com/gin-gonic/gin"
)

const (
	RefreshTokenCookieName   = "refresh_token"
	RefreshTokenCookiePath   = "/api/auth"
	RefreshTokenCookieMaxAge = 7 * 24 * 3600 // 7 days in seconds
)

func isSecureCookie(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if strings.EqualFold(c.Request.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	host := c.Request.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}
	return app.Build == app.BuildProd
}

// SetRefreshTokenCookie sets the refresh token in an HttpOnly, SameSite=Lax cookie.
func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		RefreshTokenCookieName,
		refreshToken,
		RefreshTokenCookieMaxAge,
		RefreshTokenCookiePath,
		"",
		isSecureCookie(c),
		true,
	)
}

// ClearRefreshTokenCookie clears the refresh token cookie.
func ClearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		RefreshTokenCookieName,
		"",
		-1,
		RefreshTokenCookiePath,
		"",
		isSecureCookie(c),
		true,
	)
}
