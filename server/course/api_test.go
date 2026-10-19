package course

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type mockCourseService struct {
	createCourseFunc         func(c *Course, creatorUuid uuid.UUID) (*Course, error)
	getCourseFunc            func(courseUuid uuid.UUID) (*Course, error)
	listCoursesFunc          func(term *Term, year *int) ([]Course, error)
	updateCourseFunc         func(courseUuid uuid.UUID, updated *Course) (*Course, error)
	deleteCourseFunc         func(courseUuid uuid.UUID) error
	selfEnrollFunc           func(courseUuid uuid.UUID, userUuid uuid.UUID) (*Enrollment, error)
	enrollByEmailFunc        func(courseUuid uuid.UUID, email string, role CourseRole) (*Enrollment, error)
	unenrollFunc             func(courseUuid uuid.UUID, userUuid uuid.UUID) error
	getCourseEnrollmentsFunc func(courseUuid uuid.UUID) ([]Enrollment, error)
	getUserEnrollmentsFunc   func(userUuid uuid.UUID) ([]Enrollment, error)
	isTeacherOrAdminFunc     func(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
	createModuleFunc         func(courseUuid uuid.UUID, m *Module) (*Module, error)
	getModuleFunc            func(courseUuid uuid.UUID, moduleID uint) (*Module, error)
	listModulesFunc          func(courseUuid uuid.UUID) ([]Module, error)
	updateModuleFunc         func(courseUuid uuid.UUID, moduleID uint, updated *Module) (*Module, error)
	deleteModuleFunc         func(courseUuid uuid.UUID, moduleID uint) error
	reorderModulesFunc       func(courseUuid uuid.UUID, moduleIDs []uint) error
}

func (m *mockCourseService) CreateCourse(c *Course, creatorUuid uuid.UUID) (*Course, error) {
	if m.createCourseFunc != nil {
		return m.createCourseFunc(c, creatorUuid)
	}
	c.ID = 1
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	return c, nil
}

func (m *mockCourseService) GetCourse(courseUuid uuid.UUID) (*Course, error) {
	if m.getCourseFunc != nil {
		return m.getCourseFunc(courseUuid)
	}
	return &Course{Model: gorm.Model{ID: 1}, Uuid: courseUuid, Title: "Test Course"}, nil
}

func (m *mockCourseService) ListCourses(term *Term, year *int) ([]Course, error) {
	if m.listCoursesFunc != nil {
		return m.listCoursesFunc(term, year)
	}
	return []Course{{Model: gorm.Model{ID: 1}, Uuid: uuid.New(), Title: "Test Course"}}, nil
}

func (m *mockCourseService) UpdateCourse(courseUuid uuid.UUID, updated *Course) (*Course, error) {
	if m.updateCourseFunc != nil {
		return m.updateCourseFunc(courseUuid, updated)
	}
	updated.ID = 1
	updated.Uuid = courseUuid
	return updated, nil
}

func (m *mockCourseService) DeleteCourse(courseUuid uuid.UUID) error {
	if m.deleteCourseFunc != nil {
		return m.deleteCourseFunc(courseUuid)
	}
	return nil
}

func (m *mockCourseService) SelfEnroll(courseUuid uuid.UUID, userUuid uuid.UUID) (*Enrollment, error) {
	if m.selfEnrollFunc != nil {
		return m.selfEnrollFunc(courseUuid, userUuid)
	}
	return &Enrollment{Model: gorm.Model{ID: 10}, CourseID: 1, Role: RoleStudent}, nil
}

func (m *mockCourseService) EnrollByEmail(courseUuid uuid.UUID, email string, role CourseRole) (*Enrollment, error) {
	if m.enrollByEmailFunc != nil {
		return m.enrollByEmailFunc(courseUuid, email, role)
	}
	return &Enrollment{Model: gorm.Model{ID: 11}, CourseID: 1, UserEmail: email, Role: role}, nil
}

func (m *mockCourseService) Unenroll(courseUuid uuid.UUID, userUuid uuid.UUID) error {
	if m.unenrollFunc != nil {
		return m.unenrollFunc(courseUuid, userUuid)
	}
	return nil
}

func (m *mockCourseService) GetCourseEnrollments(courseUuid uuid.UUID) ([]Enrollment, error) {
	if m.getCourseEnrollmentsFunc != nil {
		return m.getCourseEnrollmentsFunc(courseUuid)
	}
	return []Enrollment{{Model: gorm.Model{ID: 1}, CourseID: 1}}, nil
}

func (m *mockCourseService) GetUserEnrollments(userUuid uuid.UUID) ([]Enrollment, error) {
	if m.getUserEnrollmentsFunc != nil {
		return m.getUserEnrollmentsFunc(userUuid)
	}
	return []Enrollment{{
		Model:      gorm.Model{ID: 1},
		Role:       RoleStudent,
		Course:     &Course{Model: gorm.Model{ID: 1}, Uuid: uuid.New(), Title: "Enrolled Course"},
		EnrolledAt: time.Now(),
	}}, nil
}

func (m *mockCourseService) IsTeacherOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if m.isTeacherOrAdminFunc != nil {
		return m.isTeacherOrAdminFunc(courseUuid, userUuid, isSystemAdmin)
	}
	return true, nil
}

func (m *mockCourseService) CreateModule(courseUuid uuid.UUID, mod *Module) (*Module, error) {
	if m.createModuleFunc != nil {
		return m.createModuleFunc(courseUuid, mod)
	}
	mod.ID = 100
	mod.CourseID = 1
	return mod, nil
}

func (m *mockCourseService) GetModule(courseUuid uuid.UUID, moduleID uint) (*Module, error) {
	if m.getModuleFunc != nil {
		return m.getModuleFunc(courseUuid, moduleID)
	}
	return &Module{Model: gorm.Model{ID: moduleID}, CourseID: 1, Title: "Test Module"}, nil
}

func (m *mockCourseService) ListModules(courseUuid uuid.UUID) ([]Module, error) {
	if m.listModulesFunc != nil {
		return m.listModulesFunc(courseUuid)
	}
	return []Module{{Model: gorm.Model{ID: 100}, CourseID: 1, Title: "Test Module"}}, nil
}

func (m *mockCourseService) UpdateModule(courseUuid uuid.UUID, moduleID uint, updated *Module) (*Module, error) {
	if m.updateModuleFunc != nil {
		return m.updateModuleFunc(courseUuid, moduleID, updated)
	}
	updated.ID = moduleID
	updated.CourseID = 1
	return updated, nil
}

func (m *mockCourseService) DeleteModule(courseUuid uuid.UUID, moduleID uint) error {
	if m.deleteModuleFunc != nil {
		return m.deleteModuleFunc(courseUuid, moduleID)
	}
	return nil
}

func (m *mockCourseService) ReorderModules(courseUuid uuid.UUID, moduleIDs []uint) error {
	if m.reorderModulesFunc != nil {
		return m.reorderModulesFunc(courseUuid, moduleIDs)
	}
	return nil
}

func setupCourseTestRouter(service ICourseService) (*gin.Engine, *CourseCtn) {
	gin.SetMode(gin.TestMode)
	app.AccessKey = "test-course-secret-access-key-12345"
	app.RefreshKey = "test-course-secret-refresh-key-12345"

	logger := zap.NewNop().Sugar()
	ctn := &CourseCtn{
		CourseService: service,
		logger:        logger,
	}

	r := gin.New()
	api := r.Group("/api")
	ctn.RegisterEndpoints(api)
	return r, ctn
}

func generateAuthHeader(t *testing.T, userUuid uuid.UUID, role authutils.Role) string {
	token, _, err := authutils.GenerateTokens("user@hvl.no", "testuser", role, userUuid)
	assert.NoError(t, err)
	return "Bearer " + token
}

func TestCourseCtn_ListAndGet(t *testing.T) {
	r, _ := setupCourseTestRouter(&mockCourseService{})
	courseUuid := uuid.New()

	t.Run("GET /api/course returns list of courses", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/course?term=AUTUMN&year=2026", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var courses []CourseDto
		err := json.Unmarshal(w.Body.Bytes(), &courses)
		assert.NoError(t, err)
		assert.Len(t, courses, 1)
		assert.Equal(t, uint(1), courses[0].ID)
	})

	t.Run("GET /api/course/:uuid returns course", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var course CourseDto
		err := json.Unmarshal(w.Body.Bytes(), &course)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), course.ID)
		assert.Equal(t, courseUuid.String(), course.Uuid)
	})

	t.Run("GET /api/course/:uuid with invalid UUID returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/course/invalid-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /api/course/:uuid not found returns 404", func(t *testing.T) {
		service := &mockCourseService{
			getCourseFunc: func(cUuid uuid.UUID) (*Course, error) {
				return nil, ErrCourseNotFound
			},
		}
		router, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodGet, "/api/course/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCourseCtn_Create(t *testing.T) {
	service := &mockCourseService{}
	r, _ := setupCourseTestRouter(service)
	userUuid := uuid.New()
	authHeader := generateAuthHeader(t, userUuid, authutils.ROLE_USER)

	t.Run("POST /api/course without token returns 401", func(t *testing.T) {
		body, _ := json.Marshal(CreateCourseDto{
			Title: "Test",
			Term:  "AUTUMN",
			Year:  2026,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/course", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /api/course with valid token creates course", func(t *testing.T) {
		body, _ := json.Marshal(CreateCourseDto{
			Title:       "Software Technologies",
			Description: "DAT250 description",
			Term:        "AUTUMN",
			Year:        2026,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/course", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var course CourseDto
		err := json.Unmarshal(w.Body.Bytes(), &course)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), course.ID)
		assert.Equal(t, "Software Technologies", course.Title)
	})

	t.Run("POST /api/course with invalid body returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/course", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCourseCtn_UpdateAndDelete(t *testing.T) {
	userUuid := uuid.New()
	courseUuid := uuid.New()
	authHeader := generateAuthHeader(t, userUuid, authutils.ROLE_USER)

	t.Run("PUT /api/course/:uuid success for teacher", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(UpdateCourseDto{
			Title: "Updated Title",
			Term:  "SPRING",
			Year:  2027,
		})
		req, _ := http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String(), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("PUT /api/course/:uuid forbidden for non-teacher", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(UpdateCourseDto{
			Title: "Updated Title",
			Term:  "SPRING",
			Year:  2027,
		})
		req, _ := http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String(), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("DELETE /api/course/:uuid success for teacher", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodDelete, "/api/course/"+courseUuid.String(), nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestCourseCtn_EnrollmentEndpoints(t *testing.T) {
	userUuid := uuid.New()
	courseUuid := uuid.New()
	authHeader := generateAuthHeader(t, userUuid, authutils.ROLE_USER)

	t.Run("POST /api/course/:uuid/enroll self-enrolling succeeds", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enroll", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var enrollment EnrollmentDto
		err := json.Unmarshal(w.Body.Bytes(), &enrollment)
		assert.NoError(t, err)
		assert.Equal(t, RoleStudent, enrollment.Role)
	})

	t.Run("POST /api/course/:uuid/enroll already enrolled returns 409", func(t *testing.T) {
		service := &mockCourseService{
			selfEnrollFunc: func(cUuid uuid.UUID, uUuid uuid.UUID) (*Enrollment, error) {
				return nil, ErrAlreadyEnrolled
			},
		}
		r, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enroll", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("DELETE /api/course/:uuid/enroll unenrolling succeeds", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		req, _ := http.NewRequest(http.MethodDelete, "/api/course/"+courseUuid.String()+"/enroll", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("GET /api/course/my-courses returns enrolled courses", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		req, _ := http.NewRequest(http.MethodGet, "/api/course/my-courses", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var myCourses []MyCourseDto
		err := json.Unmarshal(w.Body.Bytes(), &myCourses)
		assert.NoError(t, err)
		assert.Len(t, myCourses, 1)
		assert.Equal(t, "Enrolled Course", myCourses[0].Course.Title)
		assert.Equal(t, RoleStudent, myCourses[0].Role)
	})

	t.Run("GET /api/course/:uuid/enrollments returns list for teacher", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String()+"/enrollments", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /api/course/:uuid/enrollments adds user by email", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(AddEnrollmentDto{
			Email: "student@hvl.no",
			Role:  RoleStudent,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("POST /api/course/:uuid/enrollments handles errors (not found, conflict, internal)", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			enrollByEmailFunc: func(cUuid uuid.UUID, email string, role CourseRole) (*Enrollment, error) {
				if email == "notfound@hvl.no" {
					return nil, ErrUserNotFound
				}
				if email == "conflict@hvl.no" {
					return nil, ErrAlreadyEnrolled
				}
				return nil, errors.New("db crash")
			},
		}
		r, _ := setupCourseTestRouter(service)

		// Not found
		body, _ := json.Marshal(AddEnrollmentDto{Email: "notfound@hvl.no", Role: RoleStudent})
		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Conflict
		body, _ = json.Marshal(AddEnrollmentDto{Email: "conflict@hvl.no", Role: RoleStudent})
		req, _ = http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)

		// Internal error
		body, _ = json.Marshal(AddEnrollmentDto{Email: "error@hvl.no", Role: RoleStudent})
		req, _ = http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCourseCtn_EdgeCasesAndErrors(t *testing.T) {
	userUuid := uuid.New()
	authHeader := generateAuthHeader(t, userUuid, authutils.ROLE_USER)

	t.Run("NewCourseCtn constructor", func(t *testing.T) {
		logger := zap.NewNop().Sugar()
		ctn := NewCourseCtn(&mockCourseService{}, logger)
		assert.NotNil(t, ctn)
	})

	t.Run("GET /api/course list service error", func(t *testing.T) {
		service := &mockCourseService{
			listCoursesFunc: func(term *Term, year *int) ([]Course, error) {
				return nil, errors.New("db error")
			},
		}
		r, _ := setupCourseTestRouter(service)
		req, _ := http.NewRequest(http.MethodGet, "/api/course", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("POST /api/course create user not found or db error", func(t *testing.T) {
		serviceNotFound := &mockCourseService{
			createCourseFunc: func(c *Course, creatorUuid uuid.UUID) (*Course, error) {
				return nil, ErrUserNotFound
			},
		}
		r1, _ := setupCourseTestRouter(serviceNotFound)
		body, _ := json.Marshal(CreateCourseDto{Title: "Test", Term: "SPRING", Year: 2026})
		req1, _ := http.NewRequest(http.MethodPost, "/api/course", bytes.NewBuffer(body))
		req1.Header.Set("Authorization", authHeader)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r1.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusUnauthorized, w1.Code)

		serviceErr := &mockCourseService{
			createCourseFunc: func(c *Course, creatorUuid uuid.UUID) (*Course, error) {
				return nil, errors.New("fatal")
			},
		}
		r2, _ := setupCourseTestRouter(serviceErr)
		req2, _ := http.NewRequest(http.MethodPost, "/api/course", bytes.NewBuffer(body))
		req2.Header.Set("Authorization", authHeader)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r2.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("checkTeacherOrAdmin course not found or service error", func(t *testing.T) {
		serviceNotFound := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return false, ErrCourseNotFound
			},
		}
		r1, _ := setupCourseTestRouter(serviceNotFound)
		req1, _ := http.NewRequest(http.MethodDelete, "/api/course/"+uuid.New().String(), nil)
		req1.Header.Set("Authorization", authHeader)
		w1 := httptest.NewRecorder()
		r1.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusNotFound, w1.Code)

		serviceErr := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return false, errors.New("db error")
			},
		}
		r2, _ := setupCourseTestRouter(serviceErr)
		req2, _ := http.NewRequest(http.MethodDelete, "/api/course/"+uuid.New().String(), nil)
		req2.Header.Set("Authorization", authHeader)
		w2 := httptest.NewRecorder()
		r2.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("SelfEnroll not found and internal error", func(t *testing.T) {
		serviceNotFound := &mockCourseService{
			selfEnrollFunc: func(cUuid uuid.UUID, uUuid uuid.UUID) (*Enrollment, error) {
				return nil, ErrCourseNotFound
			},
		}
		r1, _ := setupCourseTestRouter(serviceNotFound)
		req1, _ := http.NewRequest(http.MethodPost, "/api/course/"+uuid.New().String()+"/enroll", nil)
		req1.Header.Set("Authorization", authHeader)
		w1 := httptest.NewRecorder()
		r1.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusNotFound, w1.Code)

		serviceErr := &mockCourseService{
			selfEnrollFunc: func(cUuid uuid.UUID, uUuid uuid.UUID) (*Enrollment, error) {
				return nil, errors.New("db error")
			},
		}
		r2, _ := setupCourseTestRouter(serviceErr)
		req2, _ := http.NewRequest(http.MethodPost, "/api/course/"+uuid.New().String()+"/enroll", nil)
		req2.Header.Set("Authorization", authHeader)
		w2 := httptest.NewRecorder()
		r2.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("Unenroll not found and internal error", func(t *testing.T) {
		serviceNotFound := &mockCourseService{
			unenrollFunc: func(cUuid uuid.UUID, uUuid uuid.UUID) error {
				return ErrEnrollmentNotFound
			},
		}
		r1, _ := setupCourseTestRouter(serviceNotFound)
		req1, _ := http.NewRequest(http.MethodDelete, "/api/course/"+uuid.New().String()+"/enroll", nil)
		req1.Header.Set("Authorization", authHeader)
		w1 := httptest.NewRecorder()
		r1.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusNotFound, w1.Code)

		serviceErr := &mockCourseService{
			unenrollFunc: func(cUuid uuid.UUID, uUuid uuid.UUID) error {
				return errors.New("db error")
			},
		}
		r2, _ := setupCourseTestRouter(serviceErr)
		req2, _ := http.NewRequest(http.MethodDelete, "/api/course/"+uuid.New().String()+"/enroll", nil)
		req2.Header.Set("Authorization", authHeader)
		w2 := httptest.NewRecorder()
		r2.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("GetMyCourses internal error", func(t *testing.T) {
		serviceErr := &mockCourseService{
			getUserEnrollmentsFunc: func(userUuid uuid.UUID) ([]Enrollment, error) {
				return nil, errors.New("db error")
			},
		}
		r, _ := setupCourseTestRouter(serviceErr)
		req, _ := http.NewRequest(http.MethodGet, "/api/course/my-courses", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Invalid uint route params return 400", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		// update invalid id
		req, _ := http.NewRequest(http.MethodPut, "/api/course/bad-id", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// delete invalid id
		req, _ = http.NewRequest(http.MethodDelete, "/api/course/bad-id", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// self-enroll invalid id
		req, _ = http.NewRequest(http.MethodPost, "/api/course/bad-id/enroll", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// unenroll invalid id
		req, _ = http.NewRequest(http.MethodDelete, "/api/course/bad-id/enroll", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// get enrollments invalid id
		req, _ = http.NewRequest(http.MethodGet, "/api/course/bad-id/enrollments", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// add enrollment invalid id
		req, _ = http.NewRequest(http.MethodPost, "/api/course/bad-id/enrollments", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Update invalid json or not found", func(t *testing.T) {
		courseUuid := uuid.New()
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			updateCourseFunc: func(cUuid uuid.UUID, updated *Course) (*Course, error) {
				return nil, ErrCourseNotFound
			},
		}
		r, _ := setupCourseTestRouter(service)

		// invalid json
		req, _ := http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String(), bytes.NewBufferString("not-json"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// invalid model validation (bad term)
		badBody, _ := json.Marshal(UpdateCourseDto{Title: "T", Term: "INVALID", Year: 2026})
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String(), bytes.NewBuffer(badBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// not found in service
		validBody, _ := json.Marshal(UpdateCourseDto{Title: "Valid", Term: "SPRING", Year: 2026})
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String(), bytes.NewBuffer(validBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Delete and getEnrollments not found", func(t *testing.T) {
		courseUuid := uuid.New()
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			deleteCourseFunc: func(cUuid uuid.UUID) error {
				return ErrCourseNotFound
			},
			getCourseEnrollmentsFunc: func(cUuid uuid.UUID) ([]Enrollment, error) {
				return nil, ErrCourseNotFound
			},
		}
		r, _ := setupCourseTestRouter(service)

		// delete not found
		req, _ := http.NewRequest(http.MethodDelete, "/api/course/"+courseUuid.String(), nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// get enrollments not found
		req, _ = http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String()+"/enrollments", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("AddEnrollment invalid json", func(t *testing.T) {
		courseUuid := uuid.New()
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)
		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", bytes.NewBufferString("bad-json"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("checkTeacherOrAdmin context failures", func(t *testing.T) {
		logger := zap.NewNop().Sugar()
		ctn := &CourseCtn{CourseService: &mockCourseService{}, logger: logger}

		// Context without claims
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		allowed := ctn.checkTeacherOrAdmin(c, uuid.New())
		assert.False(t, allowed)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Context with claims having invalid UUID
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		invalidClaims := &authutils.Claims{
			Role:             authutils.ROLE_USER,
			RegisteredClaims: authutils.Claims{}.RegisteredClaims,
		}
		invalidClaims.RegisteredClaims.ID = "invalid-uuid"
		c2.Set(authutils.ContextClaimsKey, invalidClaims)
		allowed2 := ctn.checkTeacherOrAdmin(c2, uuid.New())
		assert.False(t, allowed2)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("Handlers abort unauthorized when user UUID is missing in context", func(t *testing.T) {
		logger := zap.NewNop().Sugar()
		ctn := &CourseCtn{CourseService: &mockCourseService{}, logger: logger}

		// create
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctn.create(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// selfEnroll
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "uuid", Value: uuid.New().String()}}
		ctn.selfEnroll(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// unenroll
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "uuid", Value: uuid.New().String()}}
		ctn.unenroll(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// getMyCourses
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		ctn.getMyCourses(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCourseCtn_Modules(t *testing.T) {
	userUuid := uuid.New()
	courseUuid := uuid.New()
	authHeader := generateAuthHeader(t, userUuid, authutils.ROLE_USER)

	t.Run("GET /api/course/:uuid/module lists modules", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		req, _ := http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String()+"/module", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var modules []ModuleDto
		err := json.Unmarshal(w.Body.Bytes(), &modules)
		assert.NoError(t, err)
		assert.Len(t, modules, 1)
		assert.Equal(t, uint(100), modules[0].ID)
	})

	t.Run("GET /api/course/:uuid/module with bad ID returns 400", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})
		req, _ := http.NewRequest(http.MethodGet, "/api/course/bad-id/module", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /api/course/:uuid/module/:moduleId gets single module", func(t *testing.T) {
		depID := uint(50)
		service := &mockCourseService{
			getModuleFunc: func(cUuid uuid.UUID, moduleID uint) (*Module, error) {
				return &Module{
					Model:       gorm.Model{ID: moduleID},
					CourseID:    1,
					Title:       "Week 2",
					Position:    1,
					DependsOnID: &depID,
					DependsOn:   &Module{Model: gorm.Model{ID: depID}, Title: "Week 1"},
				}, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String()+"/module/100", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var mod ModuleDto
		err := json.Unmarshal(w.Body.Bytes(), &mod)
		assert.NoError(t, err)
		assert.Equal(t, uint(100), mod.ID)
		assert.NotNil(t, mod.DependsOn)
		assert.Equal(t, uint(50), mod.DependsOn.ID)
	})

	t.Run("GET /api/course/:uuid/module/:moduleId bad params return 400", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		// bad course ID
		req, _ := http.NewRequest(http.MethodGet, "/api/course/bad/module/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad module ID
		req, _ = http.NewRequest(http.MethodGet, "/api/course/"+courseUuid.String()+"/module/bad", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /api/course/:uuid/module creates module for teacher", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(CreateModuleDto{Title: "Week 1: Intro"})
		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/module", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("POST /api/course/:uuid/module bad request for prereq errors and bad json", func(t *testing.T) {
		serviceSelfPrereq := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			createModuleFunc: func(cUuid uuid.UUID, m *Module) (*Module, error) {
				return nil, ErrSelfPrerequisite
			},
		}
		r, _ := setupCourseTestRouter(serviceSelfPrereq)

		body, _ := json.Marshal(CreateModuleDto{Title: "Week 1"})
		req, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/module", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad json
		req, _ = http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/module", bytes.NewBufferString("not-json"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad course ID
		req, _ = http.NewRequest(http.MethodPost, "/api/course/bad-id/module", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /api/course/:uuid/module handles not found and internal error", func(t *testing.T) {
		serviceNotFound := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			createModuleFunc: func(cUuid uuid.UUID, m *Module) (*Module, error) {
				return nil, ErrModuleNotFound
			},
		}
		r1, _ := setupCourseTestRouter(serviceNotFound)
		body, _ := json.Marshal(CreateModuleDto{Title: "Week 1"})
		req1, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/module", bytes.NewBuffer(body))
		req1.Header.Set("Authorization", authHeader)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r1.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusNotFound, w1.Code)

		serviceErr := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			createModuleFunc: func(cUuid uuid.UUID, m *Module) (*Module, error) {
				return nil, errors.New("db error")
			},
		}
		r2, _ := setupCourseTestRouter(serviceErr)
		req2, _ := http.NewRequest(http.MethodPost, "/api/course/"+courseUuid.String()+"/module", bytes.NewBuffer(body))
		req2.Header.Set("Authorization", authHeader)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r2.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("PUT /api/course/:uuid/module/:moduleId updates module", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(UpdateModuleDto{Title: "Week 1 Renamed"})
		req, _ := http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/100", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("PUT /api/course/:uuid/module/:moduleId handles errors", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			updateModuleFunc: func(cUuid uuid.UUID, moduleID uint, updated *Module) (*Module, error) {
				if moduleID == 999 {
					return nil, ErrModuleNotFound
				}
				if moduleID == 888 {
					return nil, ErrCrossCoursePrerequisite
				}
				return nil, errors.New("db error")
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(UpdateModuleDto{Title: "Valid"})

		// bad course ID
		req, _ := http.NewRequest(http.MethodPut, "/api/course/bad/module/1", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad module ID
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/bad", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad json
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/1", bytes.NewBufferString("not-json"))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// not found
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/999", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// cross-course prereq bad request
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/888", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// internal error
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/777", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("DELETE /api/course/:uuid/module/:moduleId deletes module", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		req, _ := http.NewRequest(http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/100", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("DELETE /api/course/:uuid/module/:moduleId bad params return 400", func(t *testing.T) {
		r, _ := setupCourseTestRouter(&mockCourseService{})

		// bad course ID
		req, _ := http.NewRequest(http.MethodDelete, "/api/course/bad/module/1", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad module ID
		req, _ = http.NewRequest(http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/bad", nil)
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("PUT /api/course/:uuid/module/reorder reorders modules", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
		}
		r, _ := setupCourseTestRouter(service)

		body, _ := json.Marshal(ReorderModulesDto{ModuleIDs: []uint{2, 1}})
		req, _ := http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/reorder", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("PUT /api/course/:uuid/module/reorder handles errors", func(t *testing.T) {
		service := &mockCourseService{
			isTeacherOrAdminFunc: func(cUuid uuid.UUID, uUuid uuid.UUID, isSysAdmin bool) (bool, error) {
				return true, nil
			},
			reorderModulesFunc: func(cUuid uuid.UUID, moduleIDs []uint) error {
				if len(moduleIDs) > 0 && moduleIDs[0] == 999 {
					return ErrModuleNotFound
				}
				return errors.New("db error")
			},
		}
		r, _ := setupCourseTestRouter(service)

		// bad course ID
		body, _ := json.Marshal(ReorderModulesDto{ModuleIDs: []uint{1}})
		req, _ := http.NewRequest(http.MethodPut, "/api/course/bad/module/reorder", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// bad json
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/reorder", bytes.NewBufferString("not-json"))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// not found
		notFoundBody, _ := json.Marshal(ReorderModulesDto{ModuleIDs: []uint{999}})
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/reorder", bytes.NewBuffer(notFoundBody))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// internal error
		errBody, _ := json.Marshal(ReorderModulesDto{ModuleIDs: []uint{123}})
		req, _ = http.NewRequest(http.MethodPut, "/api/course/"+courseUuid.String()+"/module/reorder", bytes.NewBuffer(errBody))
		req.Header.Set("Authorization", authHeader)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
