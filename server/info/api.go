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
	
	router.GET("/healthz", cnt.ExtractedHealthz)

	router.GET("/ping", cnt.ExtractedPing)

	println("HEALTH CHECKUP WORKS!")
}

// ExtractedHealthz godoc
// @Summary Health check
// @Description Returns the health status of the server.
// @Tags info
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func (*InfoCtn) ExtractedHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ExtractedPing godoc
// @Summary Ping the server
// @Description Returns pong if the server is reachable.
// @Tags info
// @Produce plain
// @Success 200 {string} string "pong"
// @Router /ping [get]
func (*InfoCtn) ExtractedPing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
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
