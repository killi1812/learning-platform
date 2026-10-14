package auth

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthCtn struct {
	auth   IAuthService
	logger *zap.SugaredLogger
}

func NewAuthCtn(loginService IAuthService, logger *zap.SugaredLogger) app.Api {
	return &AuthCtn{
		auth:   loginService,
		logger: logger,
	}
}

func (c *AuthCtn) RegisterEndpoints(api *gin.RouterGroup) {
	group := api.Group("/auth")

	group.POST("/login", c.login)
	group.POST("/register", c.register)
	group.POST("/refresh", c.refreshToken)

	// Protected session endpoints
	protected := group.Group("").Use(authutils.Protect())
	protected.POST("/logout", c.logoutCurrent)
	protected.POST("/logout-all", c.logoutAll)
	protected.GET("/session", c.listSessions)
	protected.DELETE("/session/:uuid", c.revokeSession)
}

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

func setRefreshTokenCookie(c *gin.Context, refreshToken string) {
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

func clearRefreshTokenCookie(c *gin.Context) {
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

// Login godoc
//
//	@Summary		User login
//	@Description	Authenticates a user and returns access token; sets HTTP-only refresh token cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			loginDto	body		LoginDto	true	"Login credentials"
//	@Success		200			{object}	TokenDto
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Router			/auth/login [post]
func (l *AuthCtn) login(c *gin.Context) {
	var loginDto LoginDto

	if err := c.ShouldBindJSON(&loginDto); err != nil {
		l.logger.Errorf("Invalid login request err = %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login payload"})
		return
	}

	tokens, err := l.auth.Login(loginDto.Username, loginDto.Password, c.Request.UserAgent(), c.ClientIP(), loginDto.SessionId)
	if err != nil {
		l.logger.Errorf("Login failed err = %+v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	setRefreshTokenCookie(c, tokens.RefreshToken)
	tokens.RefreshToken = ""
	c.JSON(http.StatusOK, tokens)
}

// Register godoc
//
//	@Summary		User registration
//	@Description	Registers a new user and returns access token; sets HTTP-only refresh token cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			registerDto	body		RegisterDto	true	"Registration credentials"
//	@Success		201			{object}	TokenDto
//	@Failure		400			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/auth/register [post]
func (l *AuthCtn) register(c *gin.Context) {
	var registerDto RegisterDto

	if err := c.ShouldBindJSON(&registerDto); err != nil {
		l.logger.Errorf("Invalid register request err = %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration payload. Check username, email format, and password length."})
		return
	}

	tokens, err := l.auth.Register(registerDto.Username, registerDto.Email, registerDto.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		l.logger.Errorf("Registration failed err = %+v", err)
		if errors.Is(err, cerror.ErrUserAlreadyExists) || strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed. Please try again."})
		return
	}

	setRefreshTokenCookie(c, tokens.RefreshToken)
	tokens.RefreshToken = ""
	c.JSON(http.StatusCreated, tokens)
}

// Refresh godoc
//
//	@Summary		Refresh Access Token
//	@Description	Generates a new access token and rotates HTTP-only refresh token cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refreshDto	body	RefreshDto	false	"Refresh token payload (optional if cookie is present)"
//	@Security		BearerAuth
//	@Success		200	{object}	TokenDto
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Router			/auth/refresh [post]
func (l *AuthCtn) refreshToken(c *gin.Context) {
	tokenStr, err := c.Cookie(RefreshTokenCookieName)
	if err != nil || tokenStr == "" {
		var refreshDto RefreshDto
		_ = c.ShouldBindJSON(&refreshDto)
		tokenStr = refreshDto.RefreshToken
		if tokenStr == "" {
			tokenStr = c.Request.Header.Get("Authorization")
		}
	}

	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing refresh token"})
		return
	}

	tokens, err := l.auth.RefreshTokens(tokenStr)
	if err != nil {
		l.logger.Errorf("Refresh failed err = %v", err)
		clearRefreshTokenCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	setRefreshTokenCookie(c, tokens.RefreshToken)
	tokens.RefreshToken = ""
	c.JSON(http.StatusOK, tokens)
}

// LogoutCurrent godoc
//
//	@Summary		User Logout (Current Device)
//	@Description	Logs out current user session and clears refresh token cookie
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/logout [post]
func (l *AuthCtn) logoutCurrent(c *gin.Context) {
	clearRefreshTokenCookie(c)

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for logout")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, ok := authutils.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for logout")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = l.auth.LogoutCurrent(userUuid, claims.SessionUuid)
	if err != nil {
		l.logger.Errorf("Logout current failed err = %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAll godoc
//
//	@Summary		User Logout All Devices
//	@Description	Logs out all active sessions for current user and clears refresh token cookie
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/logout-all [post]
func (l *AuthCtn) logoutAll(c *gin.Context) {
	clearRefreshTokenCookie(c)

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for logout-all")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = l.auth.LogoutAll(userUuid)
	if err != nil {
		l.logger.Errorf("Logout all failed err = %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout all sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All sessions logged out successfully"})
}

// ListSessions godoc
//
//	@Summary		List Active Sessions
//	@Description	Returns all active sessions for the authenticated user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		SessionDto
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/session [get]
func (l *AuthCtn) listSessions(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for listSessions")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, ok := authutils.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for listSessions")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	sessions, err := l.auth.ListSessions(userUuid, claims.SessionUuid)
	if err != nil {
		l.logger.Errorf("List sessions failed err = %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// RevokeSession godoc
//
//	@Summary		Revoke Session
//	@Description	Revokes a specific session by its ID
//	@Tags			auth
//	@Produce		json
//	@Param			uuid	path	string	true	"Session ID (UUID)"
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/session/{uuid} [delete]
func (l *AuthCtn) revokeSession(c *gin.Context) {
	targetSessionUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for revokeSession")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = l.auth.RevokeSession(userUuid, targetSessionUuid)
	if err != nil {
		l.logger.Errorf("Revoke session failed err = %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked successfully"})
}
