package auth

import (
	"net/http"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"

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
	group.POST("/logout", authUtil.Protect(), c.logout)
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
//	@Router			/auth/login [post]
func (l *AuthCtn) login(c *gin.Context) {
	var loginDto LoginDto

	if err := c.BindJSON(&loginDto); err != nil {
		l.logger.Errorf("Invalid login request err = %+v", err)
		return
	}

	accessToken, refreshToken, err := l.auth.Login(loginDto.Username, loginDto.Password)
	if err != nil {
		l.logger.Errorf("Login failed err = %+v", err)
		c.JSON(http.StatusUnauthorized, err.Error())
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
//	@Failure		400
//	@Router			/auth/register [post]
func (l *AuthCtn) register(c *gin.Context) {
	var registerDto RegisterDto

	if err := c.BindJSON(&registerDto); err != nil {
		l.logger.Errorf("Invalid register request err = %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	accessToken, refreshToken, err := l.auth.Register(registerDto.Username, registerDto.Email, registerDto.Password)
	if err != nil {
		l.logger.Errorf("Registration failed err = %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
//	@Failure		400
//	@Failure		401
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

// Logout godoc
//
//	@Summary		User Logout
//	@Description	Logs out current user session
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200
//	@Router			/auth/logout [post]
func (l *AuthCtn) logout(c *gin.Context) {
	claims, ok := authUtil.GetClaims(c)
	if !ok || claims == nil {
		l.logger.Error("Claims not found in context for logout")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err := l.auth.Logout(claims.ID)
	if err != nil {
		l.logger.Errorf("Logout failed err = %w", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	c.AbortWithStatus(http.StatusOK)
}
