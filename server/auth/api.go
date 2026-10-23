package auth

import (
	"errors"
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

// Login godoc
//
//	@Summary		User login
//	@Description	Authenticates a user by email and password, returns access token, and sets HTTP-only refresh token cookie. sessionId is optional.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			loginDto	body		LoginDto	true	"Login credentials (sessionId is optional)"
//	@Success		200			{object}	TokenDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Router			/auth/login [post]
func (l *AuthCtn) login(c *gin.Context) {
	var loginDto LoginDto

	if err := c.ShouldBindJSON(&loginDto); err != nil {
		l.logger.Errorf("Invalid login request err = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid login payload")
		return
	}

	tokens, err := l.auth.Login(loginDto.Email, loginDto.Password, c.Request.UserAgent(), c.ClientIP(), loginDto.SessionId)
	if err != nil {
		l.logger.Errorf("Login failed err = %+v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	authutils.SetRefreshTokenCookie(c, tokens.RefreshToken)
	c.JSON(http.StatusOK, TokenDto{AccessToken: tokens.AccessToken})
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
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/auth/register [post]
func (l *AuthCtn) register(c *gin.Context) {
	var registerDto RegisterDto

	if err := c.ShouldBindJSON(&registerDto); err != nil {
		l.logger.Errorf("Invalid register request err = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid registration payload. Check username, email format, and password length.")
		return
	}

	tokens, err := l.auth.Register(registerDto.Username, registerDto.Email, registerDto.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		l.logger.Errorf("Registration failed err = %+v", err)
		if errors.Is(err, cerror.ErrUserAlreadyExists) || strings.Contains(err.Error(), "already exists") {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Username or email already exists")
			return
		}
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Registration failed. Please try again.")
		return
	}

	authutils.SetRefreshTokenCookie(c, tokens.RefreshToken)
	c.JSON(http.StatusCreated, TokenDto{AccessToken: tokens.AccessToken})
}

// Refresh godoc
//
//	@Summary		Refresh Access Token
//	@Description	Generates a new access token and rotates HTTP-only refresh token cookie
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	TokenDto
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Router			/auth/refresh [post]
func (l *AuthCtn) refreshToken(c *gin.Context) {
	tokenStr, err := c.Cookie(authutils.RefreshTokenCookieName)
	if err != nil || tokenStr == "" {
		tokenStr = c.Request.Header.Get("Authorization")
	}

	if tokenStr == "" {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Missing refresh token")
		return
	}

	tokens, err := l.auth.RefreshTokens(tokenStr)
	if err != nil {
		l.logger.Errorf("Refresh failed err = %v", err)
		authutils.ClearRefreshTokenCookie(c)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	authutils.SetRefreshTokenCookie(c, tokens.RefreshToken)
	c.JSON(http.StatusOK, TokenDto{AccessToken: tokens.AccessToken})
}

// LogoutCurrent godoc
//
//	@Summary		User Logout (Current Device)
//	@Description	Logs out current user session and clears refresh token cookie
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204	"Successfully logged out"
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/auth/logout [post]
func (l *AuthCtn) logoutCurrent(c *gin.Context) {
	authutils.ClearRefreshTokenCookie(c)

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for logout")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	claims, ok := authutils.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for logout")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = l.auth.LogoutCurrent(userUuid, claims.SessionUuid)
	if err != nil {
		l.logger.Errorf("Logout current failed err = %+v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	c.Status(http.StatusNoContent)
}

// LogoutAll godoc
//
//	@Summary		User Logout All Devices
//	@Description	Logs out all active sessions for current user and clears refresh token cookie
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204	"All sessions successfully logged out"
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/auth/logout-all [post]
func (l *AuthCtn) logoutAll(c *gin.Context) {
	authutils.ClearRefreshTokenCookie(c)

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for logout-all")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = l.auth.LogoutAll(userUuid)
	if err != nil {
		l.logger.Errorf("Logout all failed err = %+v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to logout all sessions")
		return
	}

	c.Status(http.StatusNoContent)
}

// ListSessions godoc
//
//	@Summary		List Active Sessions
//	@Description	Returns all active sessions for the authenticated user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		SessionDto
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/auth/session [get]
func (l *AuthCtn) listSessions(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for listSessions")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	claims, ok := authutils.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for listSessions")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessions, err := l.auth.ListSessions(userUuid, claims.SessionUuid)
	if err != nil {
		l.logger.Errorf("List sessions failed err = %+v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to list sessions")
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
//	@Success		204	"Session successfully revoked"
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/auth/session/{uuid} [delete]
func (l *AuthCtn) revokeSession(c *gin.Context) {
	targetSessionUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		l.logger.Error("User UUID not found in context for revokeSession")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = l.auth.RevokeSession(userUuid, targetSessionUuid)
	if err != nil {
		l.logger.Errorf("Revoke session failed err = %+v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to revoke session")
		return
	}

	c.Status(http.StatusNoContent)
}
