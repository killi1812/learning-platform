package user

import (
	"net/http"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserCtn struct {
	UserCrud IUserCrudService
	logger   *zap.SugaredLogger
}

func NewUserCtn(userService IUserCrudService, logger *zap.SugaredLogger) app.Api {
	return &UserCtn{
		UserCrud: userService,
		logger:   logger,
	}
}

func (u *UserCtn) RegisterEndpoints(api *gin.RouterGroup) {
	// create a group with the name of the router
	group := api.Group("/user")

	// Protected endpint
	group.GET("/my-data", authutils.Protect(), u.getLoggedInUser)

	// register Endpoints
	group.Use(authutils.Protect(authutils.ROLE_ADMIN))
	group.POST("/", u.create)
	group.GET("/:uuid", u.get)
	group.PUT("/:uuid", u.update)
	group.DELETE("/:uuid", u.delete)
	group.GET("/all-users", u.getAllUsersForSuperAdmin)
	group.GET("/search", u.searchUsersByName)
}

// UserExample godoc
//
//	@Summary		get user with uuid
//	@Description	get a user with uuid
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200		{object}	UserDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Param			uuid	path		string	true	"user uuid"
//	@Router			/user/{uuid} [get]
func (u *UserCtn) get(c *gin.Context) {
	userUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		u.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		return
	}

	user, err := u.UserCrud.Read(userUuid)
	if err != nil {
		u.logger.Errorf("Failed to read user %s: %v", userUuid, err)
		if ginutil.HandleServiceError(c, err, ErrRecordNotFound) {
			return
		}
	}

	uDto := UserDto{}
	c.JSON(http.StatusOK, uDto.FromModel(user))
}

// UserExample godoc
//
//	@Summary	Create new user
//	@Tags		user
//	@Produce	json
//	@Security	BearerAuth
//	@Success	201		{object}	UserDto
//	@Failure	400		{object}	ginutil.ErrorResponse
//	@Failure	500		{object}	ginutil.ErrorResponse
//	@Param		model	body		NewUserDto	true	"Data for new user"
//	@Router		/user [post]
func (u *UserCtn) create(c *gin.Context) {
	var inputDto NewUserDto
	if err := c.ShouldBindJSON(&inputDto); err != nil {
		u.logger.Errorf("Failed to bind error = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	newUser, err := inputDto.ToModel()
	if err != nil {
		u.logger.Errorf("Failed to convert NewUserDto to model: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := u.UserCrud.Create(newUser, inputDto.Password)
	if err != nil {
		u.logger.Errorf("Failed to create user: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Log the response for debugging
	responseDto := UserDto{}.FromModel(user)
	u.logger.Infof("Response DTO: %+v", responseDto)

	c.JSON(http.StatusCreated, responseDto)
}

// UserExample godoc
//
//	@Summary	Update user with new dat
//	@Tags		user
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200		{object}	UserDto
//	@Failure	400		{object}	ginutil.ErrorResponse
//	@Failure	404		{object}	ginutil.ErrorResponse
//	@Failure	500		{object}	ginutil.ErrorResponse
//	@Param		uuid	path		string	true	"uuid of user to be updated"
//	@Param		model	body		UserDto	true	"Data for updating user"
//	@Router		/user/{uuid} [put]
func (u *UserCtn) update(c *gin.Context) {
	userUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		u.logger.Errorf("Error parsing UUID = %s", c.Param("uuid"))
		return
	}

	var inputDto UserDto
	if err := c.ShouldBindJSON(&inputDto); err != nil {
		u.logger.Errorf("Failed to bind error = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	newUser, err := inputDto.ToModel()
	if err != nil {
		u.logger.Errorf("Failed to convert UserDto to model: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := u.UserCrud.Update(userUuid, newUser)
	if err != nil {
		u.logger.Errorf("Failed to update user %s: %v", userUuid, err)
		if ginutil.HandleServiceError(c, err, ErrRecordNotFound) {
			return
		}
	}

	c.JSON(http.StatusOK, inputDto.FromModel(user))
}

// UserExample  godoc
//
//	@Summary		delete user with uuid
//	@Description	delete a user with uuid
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204		"User deleted successfully"
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Param			uuid	path		string	true	"user uuid"
//	@Router			/user/{uuid} [delete]
func (u *UserCtn) delete(c *gin.Context) {
	userUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		u.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		return
	}

	err := u.UserCrud.Delete(userUuid)
	if err != nil {
		u.logger.Errorf("Failed to delete user %s: %v", userUuid, err)
		if ginutil.HandleServiceError(c, err, ErrRecordNotFound) {
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// GetLoggedInUser godoc
//
//	@Summary		Get logged-in user data
//	@Description	Fetches the currently logged-in user's data based on the JWT token
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserDto
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/user/my-data [get]
func (u *UserCtn) getLoggedInUser(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		u.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := u.UserCrud.Read(userUuid)
	if err != nil {
		u.logger.Errorf("Failed to read user %s: %v", userUuid, err)
		if ginutil.HandleServiceError(c, err, ErrRecordNotFound) {
			return
		}
	}

	userDto := UserDto{}
	c.JSON(http.StatusOK, userDto.FromModel(user))
}

// GetAllUsers godoc
//
//	@Summary		Get all users for admin
//	@Description	Fetches all users for admin
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		UserDto
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/user/all-users [get]
func (u *UserCtn) getAllUsersForSuperAdmin(c *gin.Context) {
	claims, ok := authutils.GetClaims(c)
	if !ok || claims == nil {
		u.logger.Error("Claims not found in context")
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if claims.Role != authutils.ROLE_ADMIN {
		u.logger.Warnf("Unauthorized access attempt by user with role: %s", claims.Role)
		ginutil.RespondWithError(c, http.StatusForbidden, "Forbidden: insufficient permissions")
		return
	}

	users, err := u.UserCrud.GetAllUsers()
	if err != nil {
		u.logger.Errorf("Failed to fetch users: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	userDtos := make([]UserDto, 0, len(users))
	for _, user := range users {
		itemDto := UserDto{}
		userDtos = append(userDtos, itemDto.FromModel(&user))
	}

	c.JSON(http.StatusOK, userDtos)
}

// SearchUsersByName godoc
//
//	@Summary		Search users by name
//	@Description	Performs a fuzzy search for users by first name, last name, or full name with similarity matching
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Param			query	query		string	true	"Search query"
//	@Success		200		{array}		UserDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/user/search [get]
func (u *UserCtn) searchUsersByName(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		u.logger.Warn("Search query is empty")
		ginutil.RespondWithError(c, http.StatusBadRequest, "Search query is required")
		return
	}

	u.logger.Infof("Searching users with query: %s", query)

	users, err := u.UserCrud.SearchUsersByName(query)
	if err != nil {
		u.logger.Errorf("Failed to search users: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to search users")
		return
	}

	userDtos := make([]UserDto, 0, len(users))
	for _, user := range users {
		itemDto := UserDto{}
		userDtos = append(userDtos, itemDto.FromModel(&user))
	}

	c.JSON(http.StatusOK, userDtos)
}
