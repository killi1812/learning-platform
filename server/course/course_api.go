package course

import (
	"net/http"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CourseCtn struct {
	logger *zap.SugaredLogger
}

func NewInfoApi(logger *zap.SugaredLogger) app.Api {
	return &CourseCtn{
		logger: logger,
	}
}

func (cnt *CourseCtn) RegisterEndpoints(router *gin.RouterGroup) {
	// create a group with the name of the router
	group := router.Group("/groups")

	// Protected endpint
	group.POST("/join", cnt.AddStudentToGroup)

	// register Endpoints
	// group.Use(auth.Protect(auth.ROLE_ADMIN))
	// group.POST("/", u.create)
	// group.GET("/:uuid", u.get)
	// group.PUT("/:uuid", u.update)
	// group.DELETE("/:uuid", u.delete)
	// group.GET("/all-users", u.getAllUsersForSuperAdmin)
	// group.GET("/search", u.searchUsersByName)
}

// AddStudentToGroup godoc
//
//	@Summary	Lets a student join a group if available
//	@Tags		info
//	@Produce	json
//	@Success	200	{object}	ServerInfoDto	"Information about server"
//	@Router		/groups/join [post]
func (cnt *CourseCtn) AddStudentToGroup(c *gin.Context) {
	student_name := "Daniel"
	group_id := 8
	cnt.logger.Info("Student %w is trying to join group %w", student_name, group_id)
	c.AbortWithStatus(http.StatusOK)
}
