package info

import (
	"net/http"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type InfoCtn struct {
	logger *zap.SugaredLogger
}

func NewInfoApi(logger *zap.SugaredLogger) app.Api {
	return &InfoCtn{
		logger: logger,
	}
}

func (cnt *InfoCtn) RegisterEndpoints(router *gin.RouterGroup) {
	router.GET("/info", cnt.getServerInfo)
}

// getServerInfo godoc
//
//	@Summary		Get server info
//	@Description	return information about the server build, version, etc ...
//	@Tags			info
//	@Produce		json
//	@Success		200	{object}	ServerInfoDto	"Information about server"
//	@Router			/info [get]
func (ctn *InfoCtn) getServerInfo(c *gin.Context) {
	serverInfo := ServerInfoDto{
		Build:          app.Build,
		Version:        app.Version,
		CommitHash:     app.CommitHash,
		BuildTimestamp: app.BuildTimestamp,
	}
	c.AbortWithStatusJSON(http.StatusOK, serverInfo)
}
