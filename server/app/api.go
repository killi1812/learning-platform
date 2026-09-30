package app

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

// Api represents an interface for an API struct / controller
type Api interface {
	RegisterEndpoints(router *gin.RouterGroup)
}

// RegisterApi registers an Api constructor to the dig container under the "apis" group
func RegisterApi(constructor any) {
	Provide(constructor, dig.Group("apis"))
}
