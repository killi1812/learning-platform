package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/web"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSPA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("API routes return 404 JSON", func(t *testing.T) {
		router := gin.New()
		web.RegisterSPA(router)

		req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Endpoint not found")
	})

	t.Run("Swagger routes return 404 JSON", func(t *testing.T) {
		router := gin.New()
		web.RegisterSPA(router)

		req := httptest.NewRequest(http.MethodGet, "/swagger/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Endpoint not found")
	})

	t.Run("Root path serves index.html", func(t *testing.T) {
		router := gin.New()
		web.RegisterSPA(router)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<html")
	})

	t.Run("Client routes fallback to index.html", func(t *testing.T) {
		router := gin.New()
		web.RegisterSPA(router)

		req := httptest.NewRequest(http.MethodGet, "/dashboard/courses/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<html")
	})
}
