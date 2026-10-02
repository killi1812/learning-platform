package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

// RegisterSPA configures the Gin router to serve the embedded frontend SPA
func RegisterSPA(router *gin.Engine) {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(subFS))

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Never handle API or Swagger routes as SPA fallback
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/swagger") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
			return
		}

		// Try serving static file if it exists in embedded dist
		trimmedPath := strings.TrimPrefix(path, "/")
		if trimmedPath != "" {
			if f, err := subFS.Open(trimmedPath); err == nil {
				stat, statErr := f.Stat()
				_ = f.Close()
				if statErr == nil && !stat.IsDir() {
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
		}

		// Fallback to index.html for client-side routing
		indexFile, err := subFS.Open("index.html")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Frontend not found"})
			return
		}
		_ = indexFile.Close()

		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
