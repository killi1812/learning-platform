package grade

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type internalMockCourseService struct {
	course.ICourseService
}

func (m *internalMockCourseService) IsTeacherOrAdmin(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	return true, nil
}

func TestGradeCtn_UnauthenticatedContexts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctn := &GradeCtn{
		Service:       &GradeService{},
		courseService: &internalMockCourseService{},
		logger:        zap.NewNop().Sugar(),
	}

	t.Run("upsertGrade returns 401 when user uuid missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{
			{Key: "uuid", Value: uuid.New().String()},
			{Key: "contentUuid", Value: uuid.New().String()},
			{Key: "submissionUuid", Value: uuid.New().String()},
		}
		c.Request, _ = http.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"score": 50}`))
		c.Request.Header.Set("Content-Type", "application/json")

		ctn.upsertGrade(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("listPending returns 401 when user uuid missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

		ctn.listPending(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("listMyGrades returns 401 when user uuid missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

		ctn.listMyGrades(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
