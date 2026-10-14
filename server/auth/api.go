package auth

import (
	"errors"
	"net/http"
	"strings"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	group.POST("/logout", authUtil.Protect(), c.logoutCurrent)
	group.POST("/logout-all", authUtil.Protect(), c.logoutAll)
	group.GET("/sessions", authUtil.Protect(), c.listSessions)
	group.DELETE("/sessions/:id", authUtil.Protect(), c.revokeSession)
}

// Login godoc
//
//	@Summary		User login
//	@Description	Authenticates a user and returns access and refresh tokens
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

	accessToken, refreshToken, err := l.auth.Login(loginDto.Username, loginDto.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		l.logger.Errorf("Login failed err = %+v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, TokenDto{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Register godoc
//
//	@Summary		User registration
//	@Description	Registers a new user and returns access and refresh tokens
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

	accessToken, refreshToken, err := l.auth.Register(registerDto.Username, registerDto.Email, registerDto.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		l.logger.Errorf("Registration failed err = %+v", err)
		if errors.Is(err, cerror.ErrUserAlreadyExists) || strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed. Please try again."})
		return
	}

	c.JSON(http.StatusCreated, TokenDto{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Refresh godoc
//
//	@Summary		Refresh Access Token
//	@Description	Generates a new access token and refresh token using a valid refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refreshDto	body	RefreshDto	false	"Refresh token payload"
//	@Security		BearerAuth
//	@Success		200	{object}	TokenDto
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Router			/auth/refresh [post]
func (l *AuthCtn) refreshToken(c *gin.Context) {
	var refreshDto RefreshDto
	_ = c.ShouldBindJSON(&refreshDto)

	tokenStr := refreshDto.RefreshToken
	if tokenStr == "" {
		tokenStr = c.Request.Header.Get("Authorization")
	}

	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing refresh token"})
		return
	}

	accessToken, newRefreshToken, err := l.auth.RefreshTokens(tokenStr)
	if err != nil {
		l.logger.Errorf("Refresh failed err = %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, TokenDto{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

// LogoutCurrent godoc
//
//	@Summary		User Logout (Current Device)
//	@Description	Logs out current user session only
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/logout [post]
func (l *AuthCtn) logoutCurrent(c *gin.Context) {
	claims, ok := authUtil.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for logout")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err := l.auth.LogoutCurrent(claims.ID, claims.SessionUuid)
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
//	@Description	Logs out all active sessions for current user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/logout-all [post]
func (l *AuthCtn) logoutAll(c *gin.Context) {
	claims, ok := authUtil.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for logout-all")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err := l.auth.LogoutAll(claims.ID)
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
//	@Router			/auth/sessions [get]
func (l *AuthCtn) listSessions(c *gin.Context) {
	claims, ok := authUtil.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for listSessions")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	sessions, err := l.auth.ListSessions(claims.ID, claims.SessionUuid)
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
//	@Param			id	path	string	true	"Session ID (UUID)"
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/sessions/{id} [delete]
func (l *AuthCtn) revokeSession(c *gin.Context) {
	claims, ok := authUtil.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for revokeSession")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	targetSessionIdStr := c.Param("id")
	targetSessionUuid, err := uuid.Parse(targetSessionIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	err = l.auth.RevokeSession(claims.ID, targetSessionUuid)
	if err != nil {
		l.logger.Errorf("Revoke session failed err = %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked successfully"})
}
