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
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
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
	withdrawFunc             func(courseUuid uuid.UUID, userUuid uuid.UUID) error
	getCourseEnrollmentsFunc func(courseUuid uuid.UUID) ([]Enrollment, error)
	getUserEnrollmentsFunc   func(userUuid uuid.UUID) ([]Enrollment, error)
	isTeacherOrAdminFunc     func(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
	isEnrolledOrAdminFunc    func(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
	createModuleFunc         func(courseUuid uuid.UUID, m *Module) (*Module, error)
	getModuleFunc            func(courseUuid uuid.UUID, moduleUuid uuid.UUID) (*Module, error)
	listModulesFunc          func(courseUuid uuid.UUID) ([]Module, error)
	updateModuleFunc         func(courseUuid uuid.UUID, moduleUuid uuid.UUID, updated *Module) (*Module, error)
	deleteModuleFunc         func(courseUuid uuid.UUID, moduleUuid uuid.UUID) error
	reorderModulesFunc       func(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error
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

func (m *mockCourseService) Withdraw(courseUuid uuid.UUID, userUuid uuid.UUID) error {
	if m.withdrawFunc != nil {
		return m.withdrawFunc(courseUuid, userUuid)
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

func (m *mockCourseService) IsEnrolledOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if m.isEnrolledOrAdminFunc != nil {
		return m.isEnrolledOrAdminFunc(courseUuid, userUuid, isSystemAdmin)
	}
	return true, nil
}

func (m *mockCourseService) CreateModule(courseUuid uuid.UUID, mod *Module) (*Module, error) {
	if m.createModuleFunc != nil {
		return m.createModuleFunc(courseUuid, mod)
	}
	mod.ID = 100
	if mod.Uuid == uuid.Nil {
		mod.Uuid = uuid.New()
	}
	mod.CourseID = 1
	return mod, nil
}

func (m *mockCourseService) GetModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) (*Module, error) {
	if m.getModuleFunc != nil {
		return m.getModuleFunc(courseUuid, moduleUuid)
	}
	return &Module{Model: gorm.Model{ID: 100}, Uuid: moduleUuid, CourseID: 1, Title: "Test Module"}, nil
}

func (m *mockCourseService) ListModules(courseUuid uuid.UUID) ([]Module, error) {
	if m.listModulesFunc != nil {
		return m.listModulesFunc(courseUuid)
	}
	return []Module{{Model: gorm.Model{ID: 100}, Uuid: uuid.New(), CourseID: 1, Title: "Test Module"}}, nil
}

func (m *mockCourseService) UpdateModule(courseUuid uuid.UUID, moduleUuid uuid.UUID, updated *Module) (*Module, error) {
	if m.updateModuleFunc != nil {
		return m.updateModuleFunc(courseUuid, moduleUuid, updated)
	}
	updated.ID = 100
	updated.Uuid = moduleUuid
	updated.CourseID = 1
	return updated, nil
}

func (m *mockCourseService) DeleteModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) error {
	if m.deleteModuleFunc != nil {
		return m.deleteModuleFunc(courseUuid, moduleUuid)
	}
	return nil
}

func (m *mockCourseService) ReorderModules(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error {
	if m.reorderModulesFunc != nil {
		return m.reorderModulesFunc(courseUuid, moduleUuids)
	}
	return nil
}

type mockGroupService struct {
	listCourseGroupsFunc       func(courseUuid uuid.UUID) ([]Group, error)
	getGroupFunc               func(groupUuid uuid.UUID) (*Group, error)
	createGroupFunc            func(courseUuid uuid.UUID, name string, maxMembers int) (*Group, error)
	bulkCreateGroupsFunc       func(courseUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error)
	updateGroupFunc            func(groupUuid uuid.UUID, name string, maxMembers int) (*Group, error)
	deleteGroupFunc            func(groupUuid uuid.UUID) error
	userJoinGroupFunc          func(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error)
	removeStudentFromGroupFunc func(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error)
}

func (m *mockGroupService) ListCourseGroups(courseUuid uuid.UUID) ([]Group, error) {
	if m.listCourseGroupsFunc != nil {
		return m.listCourseGroupsFunc(courseUuid)
	}
	return []Group{{Name: "Group 1", MaxMembers: 4}}, nil
}

func (m *mockGroupService) GetGroup(groupUuid uuid.UUID) (*Group, error) {
	if m.getGroupFunc != nil {
		return m.getGroupFunc(groupUuid)
	}
	return &Group{Uuid: groupUuid, Name: "Group 1", MaxMembers: 4, Course: &Course{Uuid: uuid.New()}}, nil
}

func (m *mockGroupService) CreateGroup(courseUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
	if m.createGroupFunc != nil {
		return m.createGroupFunc(courseUuid, name, maxMembers)
	}
	return &Group{Uuid: uuid.New(), Name: name, MaxMembers: maxMembers, Course: &Course{Uuid: courseUuid}}, nil
}

func (m *mockGroupService) BulkCreateGroups(courseUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error) {
	if m.bulkCreateGroupsFunc != nil {
		return m.bulkCreateGroupsFunc(courseUuid, prefix, count, maxMembers)
	}
	res := make([]Group, count)
	for i := 0; i < count; i++ {
		res[i] = Group{Uuid: uuid.New(), Name: "G", MaxMembers: maxMembers, Course: &Course{Uuid: courseUuid}}
	}
	return res, nil
}

func (m *mockGroupService) UpdateGroup(groupUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
	if m.updateGroupFunc != nil {
		return m.updateGroupFunc(groupUuid, name, maxMembers)
	}
	return &Group{Uuid: groupUuid, Name: name, MaxMembers: maxMembers, Course: &Course{Uuid: uuid.New()}}, nil
}

func (m *mockGroupService) DeleteGroup(groupUuid uuid.UUID) error {
	if m.deleteGroupFunc != nil {
		return m.deleteGroupFunc(groupUuid)
	}
	return nil
}

func (m *mockGroupService) UserJoinGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error) {
	if m.userJoinGroupFunc != nil {
		return m.userJoinGroupFunc(groupUuid, userUuid)
	}
	return &Group{Uuid: groupUuid, Name: "Group 1", MaxMembers: 4, Course: &Course{Uuid: uuid.New()}}, nil
}

func (m *mockGroupService) RemoveStudentFromGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error) {
	if m.removeStudentFromGroupFunc != nil {
		return m.removeStudentFromGroupFunc(groupUuid, userUuid)
	}
	return &Group{Uuid: groupUuid, Name: "Group 1", MaxMembers: 4, Course: &Course{Uuid: uuid.New()}}, nil
}

func init() {
	app.AccessKey = "test-course-secret-access-key-12345"
	app.RefreshKey = "test-course-secret-refresh-key-12345"
}

func setupCourseTestRouter(service ICourseService, groupSvc ...IGroupService) (*gin.Engine, *CourseCtn) {
	gin.SetMode(gin.TestMode)

	var gs IGroupService = &mockGroupService{}
	if len(groupSvc) > 0 && groupSvc[0] != nil {
		gs = groupSvc[0]
	}

	ctn := &CourseCtn{
		CourseService: service,
		groupService:  gs,
		logger:        zap.NewNop().Sugar(),
	}
	r := gin.New()
	ctn.RegisterEndpoints(r.Group("/api"))
	return r, ctn
}

func generateAuthHeader(t *testing.T, userUuid uuid.UUID, role authutils.Role) string {
	token, _, err := authutils.GenerateTokens("user@hvl.no", "testuser", role, userUuid)
	assert.NoError(t, err)
	return "Bearer " + token
}

func doReq(r http.Handler, method, path, auth string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		switch v := body.(type) {
		case string:
			buf.WriteString(v)
		default:
			_ = json.NewEncoder(&buf).Encode(v)
		}
	}
	req, _ := http.NewRequest(method, path, &buf)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCourseCtn_HappyPaths(t *testing.T) {
	assert.NotNil(t, NewCourseCtn(&mockCourseService{}, zap.NewNop().Sugar(), nil, nil))

	depID := uint(50)
	depUuid := uuid.New()
	modUuid := uuid.New()
	courseUuid := uuid.New()
	targetUserUuid := uuid.New()
	auth := generateAuthHeader(t, uuid.New(), authutils.ROLE_USER)

	svc := &mockCourseService{
		getModuleFunc: func(cUuid, mUuid uuid.UUID) (*Module, error) {
			return &Module{
				Model:       gorm.Model{ID: 100},
				Uuid:        mUuid,
				CourseID:    1,
				Title:       "Week 2",
				DependsOnID: &depID,
				DependsOn:   &Module{Model: gorm.Model{ID: depID}, Uuid: depUuid, Title: "Week 1"},
			}, nil
		},
	}
	r, _ := setupCourseTestRouter(svc)

	// List courses
	w := doReq(r, http.MethodGet, "/api/course?term=AUTUMN&year=2026", auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get course
	w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String(), auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Create course
	w = doReq(r, http.MethodPost, "/api/course", auth, CreateCourseDto{
		Title: "Software Technologies", Description: "DAT250", Term: "AUTUMN", Year: 2026,
	})
	assert.Equal(t, http.StatusCreated, w.Code)

	// Update course
	w = doReq(r, http.MethodPut, "/api/course/"+courseUuid.String(), auth, UpdateCourseDto{
		Title: "Updated", Term: "SPRING", Year: 2027,
	})
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete course
	w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String(), auth, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Self-enroll & withdraw
	w = doReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/enroll", auth, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/enroll", auth, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// My courses & course enrollments
	w = doReq(r, http.MethodGet, "/api/course/my-course", auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/enrollments", auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = doReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/enrollments", auth, AddEnrollmentDto{
		Email: "student@hvl.no", Role: RoleStudent,
	})
	assert.Equal(t, http.StatusCreated, w.Code)
	w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/enrollments/"+targetUserUuid.String(), auth, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Modules CRUD & reorder
	w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module", auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+modUuid.String(), auth, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = doReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module", auth, CreateModuleDto{Title: "Week 1"})
	assert.Equal(t, http.StatusCreated, w.Code)
	w = doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+modUuid.String(), auth, UpdateModuleDto{Title: "Week 1 Updated"})
	assert.Equal(t, http.StatusOK, w.Code)
	w = doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/reorder", auth, ReorderModulesDto{ModuleUuids: []uuid.UUID{modUuid}})
	assert.Equal(t, http.StatusNoContent, w.Code)
	w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/"+modUuid.String(), auth, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCourseCtn_ValidationAndParamErrors(t *testing.T) {
	r, _ := setupCourseTestRouter(&mockCourseService{})
	auth := generateAuthHeader(t, uuid.New(), authutils.ROLE_USER)
	cID := uuid.New().String()
	mID := uuid.New().String()
	uID := uuid.New().String()

	cases := []struct {
		method string
		path   string
		auth   string
		body   any
		want   int
	}{
		{http.MethodPost, "/api/course", "", CreateCourseDto{Title: "T", Term: "AUTUMN", Year: 2026}, http.StatusUnauthorized},
		{http.MethodPost, "/api/course", auth, "{bad", http.StatusBadRequest},
		{http.MethodPost, "/api/course", auth, CreateCourseDto{Title: "T", Term: "INVALID", Year: 2026}, http.StatusBadRequest},
		{http.MethodGet, "/api/course/bad-uuid", auth, nil, http.StatusBadRequest},
		{http.MethodPut, "/api/course/bad-uuid", auth, "{}", http.StatusBadRequest},
		{http.MethodPut, "/api/course/" + cID, auth, "{bad", http.StatusBadRequest},
		{http.MethodPut, "/api/course/" + cID, auth, UpdateCourseDto{Title: "T", Term: "INVALID", Year: 2026}, http.StatusBadRequest},
		{http.MethodDelete, "/api/course/bad-uuid", auth, nil, http.StatusBadRequest},
		{http.MethodPost, "/api/course/bad-uuid/enroll", auth, nil, http.StatusBadRequest},
		{http.MethodDelete, "/api/course/bad-uuid/enroll", auth, nil, http.StatusBadRequest},
		{http.MethodGet, "/api/course/bad-uuid/enrollments", auth, nil, http.StatusBadRequest},
		{http.MethodPost, "/api/course/bad-uuid/enrollments", auth, "{}", http.StatusBadRequest},
		{http.MethodPost, "/api/course/" + cID + "/enrollments", auth, "{bad", http.StatusBadRequest},
		{http.MethodDelete, "/api/course/bad-uuid/enrollments/" + uID, auth, nil, http.StatusBadRequest},
		{http.MethodDelete, "/api/course/" + cID + "/enrollments/bad-uuid", auth, nil, http.StatusBadRequest},
		{http.MethodGet, "/api/course/bad-uuid/module", auth, nil, http.StatusBadRequest},
		{http.MethodGet, "/api/course/bad-uuid/module/" + mID, auth, nil, http.StatusBadRequest},
		{http.MethodGet, "/api/course/" + cID + "/module/bad-uuid", auth, nil, http.StatusBadRequest},
		{http.MethodPost, "/api/course/bad-uuid/module", auth, CreateModuleDto{Title: "W1"}, http.StatusBadRequest},
		{http.MethodPost, "/api/course/" + cID + "/module", auth, "{bad", http.StatusBadRequest},
		{http.MethodPut, "/api/course/bad-uuid/module/" + mID, auth, UpdateModuleDto{Title: "W1"}, http.StatusBadRequest},
		{http.MethodPut, "/api/course/" + cID + "/module/bad-uuid", auth, UpdateModuleDto{Title: "W1"}, http.StatusBadRequest},
		{http.MethodPut, "/api/course/" + cID + "/module/" + mID, auth, "{bad", http.StatusBadRequest},
		{http.MethodDelete, "/api/course/bad-uuid/module/" + mID, auth, nil, http.StatusBadRequest},
		{http.MethodDelete, "/api/course/" + cID + "/module/bad-uuid", auth, nil, http.StatusBadRequest},
		{http.MethodPut, "/api/course/bad-uuid/module/reorder", auth, ReorderModulesDto{ModuleUuids: []uuid.UUID{uuid.New()}}, http.StatusBadRequest},
		{http.MethodPut, "/api/course/" + cID + "/module/reorder", auth, "{bad", http.StatusBadRequest},
	}

	for _, tc := range cases {
		w := doReq(r, tc.method, tc.path, tc.auth, tc.body)
		assert.Equal(t, tc.want, w.Code, "%s %s", tc.method, tc.path)
	}
}

func TestCourseCtn_ServiceErrorsAndPermissions(t *testing.T) {
	auth := generateAuthHeader(t, uuid.New(), authutils.ROLE_USER)
	cID := uuid.New().String()
	mID := uuid.New().String()
	uID := uuid.New().String()
	dbErr := errors.New("db error")

	// Forbidden when not teacher/admin
	rForbidden, _ := setupCourseTestRouter(&mockCourseService{
		isTeacherOrAdminFunc: func(uuid.UUID, uuid.UUID, bool) (bool, error) { return false, nil },
	})
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "/api/course/" + cID, UpdateCourseDto{Title: "Title", Term: "SPRING", Year: 2026}},
		{http.MethodDelete, "/api/course/" + cID, nil},
		{http.MethodGet, "/api/course/" + cID + "/enrollments", nil},
		{http.MethodPost, "/api/course/" + cID + "/enrollments", AddEnrollmentDto{Email: "a@b.c"}},
		{http.MethodDelete, "/api/course/" + cID + "/enrollments/" + uID, nil},
		{http.MethodPost, "/api/course/" + cID + "/module", CreateModuleDto{Title: "Module"}},
		{http.MethodPut, "/api/course/" + cID + "/module/" + mID, UpdateModuleDto{Title: "Module"}},
		{http.MethodDelete, "/api/course/" + cID + "/module/" + mID, nil},
		{http.MethodPut, "/api/course/" + cID + "/module/reorder", ReorderModulesDto{ModuleUuids: []uuid.UUID{uuid.New()}}},
	} {
		assert.Equal(t, http.StatusForbidden, doReq(rForbidden, tc.method, tc.path, auth, tc.body).Code)
	}

	// Forbidden when not enrolled/admin for modules
	rNotEnrolled, _ := setupCourseTestRouter(&mockCourseService{
		isEnrolledOrAdminFunc: func(uuid.UUID, uuid.UUID, bool) (bool, error) { return false, nil },
	})
	assert.Equal(t, http.StatusForbidden, doReq(rNotEnrolled, http.MethodGet, "/api/course/"+cID+"/module", auth, nil).Code)
	assert.Equal(t, http.StatusForbidden, doReq(rNotEnrolled, http.MethodGet, "/api/course/"+cID+"/module/"+mID, auth, nil).Code)

	// NotFound & Conflict & BadRequest service errors
	rNotFound, _ := setupCourseTestRouter(&mockCourseService{
		getCourseFunc:            func(uuid.UUID) (*Course, error) { return nil, ErrCourseNotFound },
		createCourseFunc:         func(*Course, uuid.UUID) (*Course, error) { return nil, ErrUserNotFound },
		updateCourseFunc:         func(uuid.UUID, *Course) (*Course, error) { return nil, ErrCourseNotFound },
		deleteCourseFunc:         func(uuid.UUID) error { return ErrCourseNotFound },
		selfEnrollFunc:           func(uuid.UUID, uuid.UUID) (*Enrollment, error) { return nil, ErrAlreadyEnrolled },
		withdrawFunc:             func(uuid.UUID, uuid.UUID) error { return ErrEnrollmentNotFound },
		getCourseEnrollmentsFunc: func(uuid.UUID) ([]Enrollment, error) { return nil, ErrCourseNotFound },
		enrollByEmailFunc:        func(uuid.UUID, string, CourseRole) (*Enrollment, error) { return nil, ErrUserNotFound },
		listModulesFunc:          func(uuid.UUID) ([]Module, error) { return nil, ErrCourseNotFound },
		getModuleFunc:            func(uuid.UUID, uuid.UUID) (*Module, error) { return nil, ErrModuleNotFound },
		createModuleFunc:         func(uuid.UUID, *Module) (*Module, error) { return nil, ErrSelfPrerequisite },
		updateModuleFunc:         func(uuid.UUID, uuid.UUID, *Module) (*Module, error) { return nil, ErrCrossCoursePrerequisite },
		deleteModuleFunc:         func(uuid.UUID, uuid.UUID) error { return ErrModuleNotFound },
		reorderModulesFunc:       func(uuid.UUID, []uuid.UUID) error { return ErrModuleNotFound },
	})
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodGet, "/api/course/"+cID, auth, nil).Code)
	assert.Equal(t, http.StatusUnauthorized, doReq(rNotFound, http.MethodPost, "/api/course", auth, CreateCourseDto{Title: "Title", Term: "SPRING", Year: 2026}).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodPut, "/api/course/"+cID, auth, UpdateCourseDto{Title: "Title", Term: "SPRING", Year: 2026}).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodDelete, "/api/course/"+cID, auth, nil).Code)
	assert.Equal(t, http.StatusConflict, doReq(rNotFound, http.MethodPost, "/api/course/"+cID+"/enroll", auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodDelete, "/api/course/"+cID+"/enroll", auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodGet, "/api/course/"+cID+"/enrollments", auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodPost, "/api/course/"+cID+"/enrollments", auth, AddEnrollmentDto{Email: "a@b.c"}).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodDelete, "/api/course/"+cID+"/enrollments/"+uID, auth, nil).Code)

	// Cannot remove teacher returns 400 BadRequest
	rCannotRemoveTeacher, _ := setupCourseTestRouter(&mockCourseService{
		withdrawFunc: func(uuid.UUID, uuid.UUID) error { return ErrCannotRemoveTeacher },
	})
	assert.Equal(t, http.StatusBadRequest, doReq(rCannotRemoveTeacher, http.MethodDelete, "/api/course/"+cID+"/enrollments/"+uID, auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodGet, "/api/course/"+cID+"/module", auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodGet, "/api/course/"+cID+"/module/"+mID, auth, nil).Code)
	assert.Equal(t, http.StatusBadRequest, doReq(rNotFound, http.MethodPost, "/api/course/"+cID+"/module", auth, CreateModuleDto{Title: "Module"}).Code)
	assert.Equal(t, http.StatusBadRequest, doReq(rNotFound, http.MethodPut, "/api/course/"+cID+"/module/"+mID, auth, UpdateModuleDto{Title: "Module"}).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodDelete, "/api/course/"+cID+"/module/"+mID, auth, nil).Code)
	assert.Equal(t, http.StatusNotFound, doReq(rNotFound, http.MethodPut, "/api/course/"+cID+"/module/reorder", auth, ReorderModulesDto{ModuleUuids: []uuid.UUID{uuid.New()}}).Code)

	// 500 Internal Server Errors
	rInternal, _ := setupCourseTestRouter(&mockCourseService{
		listCoursesFunc:        func(*Term, *int) ([]Course, error) { return nil, dbErr },
		createCourseFunc:       func(*Course, uuid.UUID) (*Course, error) { return nil, dbErr },
		selfEnrollFunc:         func(uuid.UUID, uuid.UUID) (*Enrollment, error) { return nil, dbErr },
		withdrawFunc:           func(uuid.UUID, uuid.UUID) error { return dbErr },
		getUserEnrollmentsFunc: func(uuid.UUID) ([]Enrollment, error) { return nil, dbErr },
		enrollByEmailFunc:      func(uuid.UUID, string, CourseRole) (*Enrollment, error) { return nil, dbErr },
		createModuleFunc:       func(uuid.UUID, *Module) (*Module, error) { return nil, dbErr },
		updateModuleFunc:       func(uuid.UUID, uuid.UUID, *Module) (*Module, error) { return nil, dbErr },
		reorderModulesFunc:     func(uuid.UUID, []uuid.UUID) error { return dbErr },
	})
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodGet, "/api/course", auth, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPost, "/api/course", auth, CreateCourseDto{Title: "Title", Term: "SPRING", Year: 2026}).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPost, "/api/course/"+cID+"/enroll", auth, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodDelete, "/api/course/"+cID+"/enroll", auth, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodGet, "/api/course/my-course", auth, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPost, "/api/course/"+cID+"/enrollments", auth, AddEnrollmentDto{Email: "a@b.c"}).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodDelete, "/api/course/"+cID+"/enrollments/"+uID, auth, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPost, "/api/course/"+cID+"/module", auth, CreateModuleDto{Title: "Module"}).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPut, "/api/course/"+cID+"/module/"+mID, auth, UpdateModuleDto{Title: "Module"}).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(rInternal, http.MethodPut, "/api/course/"+cID+"/module/reorder", auth, ReorderModulesDto{ModuleUuids: []uuid.UUID{uuid.New()}}).Code)

	// Direct handler calls without claims in context
	ctn := &CourseCtn{CourseService: &mockCourseService{}, logger: zap.NewNop().Sugar()}
	for _, fn := range []func(*gin.Context){ctn.create, ctn.selfEnroll, ctn.withdraw, ctn.getMyCourses} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "uuid", Value: cID}}
		fn(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}
}

func TestCourseCtn_GroupEndpoints(t *testing.T) {
	userUuid := uuid.New()
	auth := generateAuthHeader(t, userUuid, authutils.ROLE_USER)
	courseUuid := uuid.New()
	cID := courseUuid.String()
	groupUuid := uuid.New()
	gID := groupUuid.String()

	courseSvc := &mockCourseService{
		isEnrolledOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
			return true, nil
		},
		isTeacherOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
			return true, nil
		},
	}

	groupSvc := &mockGroupService{
		listCourseGroupsFunc: func(cUuid uuid.UUID) ([]Group, error) {
			return []Group{
				{
					Uuid:       groupUuid,
					CourseID:   1,
					Course:     &Course{Uuid: cUuid},
					Name:       "Test Group 1",
					MaxMembers: 4,
					Members: []user.User{
						{Uuid: userUuid, Username: "student1", Email: "s1@example.com"},
					},
				},
			}, nil
		},
		getGroupFunc: func(gUuid uuid.UUID) (*Group, error) {
			return &Group{
				Uuid:       gUuid,
				CourseID:   1,
				Course:     &Course{Uuid: courseUuid},
				Name:       "Test Group 1",
				MaxMembers: 4,
				Members:    []user.User{},
			}, nil
		},
		createGroupFunc: func(cUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
			return &Group{
				Uuid:       uuid.New(),
				CourseID:   1,
				Course:     &Course{Uuid: cUuid},
				Name:       name,
				MaxMembers: maxMembers,
				Members:    []user.User{},
			}, nil
		},
		bulkCreateGroupsFunc: func(cUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error) {
			return []Group{
				{Uuid: uuid.New(), Course: &Course{Uuid: cUuid}, Name: prefix + " 1", MaxMembers: maxMembers},
				{Uuid: uuid.New(), Course: &Course{Uuid: cUuid}, Name: prefix + " 2", MaxMembers: maxMembers},
			}, nil
		},
		updateGroupFunc: func(gUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
			return &Group{
				Uuid:       gUuid,
				CourseID:   1,
				Course:     &Course{Uuid: courseUuid},
				Name:       name,
				MaxMembers: maxMembers,
				Members:    []user.User{},
			}, nil
		},
		deleteGroupFunc: func(gUuid uuid.UUID) error {
			return nil
		},
		userJoinGroupFunc: func(gUuid, uUuid uuid.UUID) (*Group, error) {
			return &Group{
				Uuid:       gUuid,
				CourseID:   1,
				Course:     &Course{Uuid: courseUuid},
				Name:       "Test Group 1",
				MaxMembers: 4,
				Members:    []user.User{{Uuid: uUuid, Username: "joiner"}},
			}, nil
		},
		removeStudentFromGroupFunc: func(gUuid, uUuid uuid.UUID) (*Group, error) {
			return &Group{
				Uuid:       gUuid,
				CourseID:   1,
				Course:     &Course{Uuid: courseUuid},
				Name:       "Test Group 1",
				MaxMembers: 4,
				Members:    []user.User{},
			}, nil
		},
	}

	r, _ := setupCourseTestRouter(courseSvc, groupSvc)

	t.Run("List course groups", func(t *testing.T) {
		w := doReq(r, http.MethodGet, "/api/course/"+cID+"/group", auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var dtos []GroupDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 1)
		assert.Equal(t, "Test Group 1", dtos[0].Name)

		// Invalid course UUID
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodGet, "/api/course/not-uuid/group", auth, nil).Code)
	})

	t.Run("Get group", func(t *testing.T) {
		w := doReq(r, http.MethodGet, "/api/group/"+gID, auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var dto GroupDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "Test Group 1", dto.Name)

		// Invalid group UUID
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodGet, "/api/group/not-uuid", auth, nil).Code)
	})

	t.Run("Create group", func(t *testing.T) {
		w := doReq(r, http.MethodPost, "/api/group", auth, CreateGroupDto{
			CourseUuid: cID,
			Name:       "New Group",
			MaxMembers: 4,
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		// Invalid payload
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPost, "/api/group", auth, "bad-json").Code)
		// Invalid course UUID
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPost, "/api/group", auth, CreateGroupDto{
			CourseUuid: "bad-uuid",
			Name:       "New Group",
			MaxMembers: 4,
		}).Code)
	})

	t.Run("Bulk create groups", func(t *testing.T) {
		w := doReq(r, http.MethodPost, "/api/group/bulk", auth, BulkCreateGroupsDto{
			CourseUuid: cID,
			Prefix:     "Team",
			Count:      2,
			MaxMembers: 4,
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		var dtos []GroupDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 2)

		// Invalid payload
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPost, "/api/group/bulk", auth, "bad-json").Code)
		// Invalid course UUID
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPost, "/api/group/bulk", auth, BulkCreateGroupsDto{
			CourseUuid: "bad-uuid",
			Count:      2,
			MaxMembers: 4,
		}).Code)
	})

	t.Run("Update group", func(t *testing.T) {
		w := doReq(r, http.MethodPatch, "/api/group/"+gID, auth, UpdateGroupDto{
			Name:       "Renamed",
			MaxMembers: 6,
		})
		assert.Equal(t, http.StatusOK, w.Code)

		// Invalid uuid
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPatch, "/api/group/not-uuid", auth, UpdateGroupDto{
			Name:       "Renamed",
			MaxMembers: 6,
		}).Code)
		// Invalid payload
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPatch, "/api/group/"+gID, auth, "bad-json").Code)
	})

	t.Run("Delete group", func(t *testing.T) {
		w := doReq(r, http.MethodDelete, "/api/group/"+gID, auth, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)

		// Invalid uuid
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodDelete, "/api/group/not-uuid", auth, nil).Code)
	})

	t.Run("Join and leave group", func(t *testing.T) {
		w := doReq(r, http.MethodPost, "/api/group/join/"+gID, auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Without auth header
		assert.Equal(t, http.StatusUnauthorized, doReq(r, http.MethodPost, "/api/group/join/"+gID, "", nil).Code)
		// Invalid uuid
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPost, "/api/group/join/not-uuid", auth, nil).Code)

		// Leave
		wLeave := doReq(r, http.MethodPatch, "/api/group/leave/"+gID, auth, nil)
		assert.Equal(t, http.StatusOK, wLeave.Code)

		// Leave without auth header
		assert.Equal(t, http.StatusUnauthorized, doReq(r, http.MethodPatch, "/api/group/leave/"+gID, "", nil).Code)
		// Leave invalid uuid
		assert.Equal(t, http.StatusBadRequest, doReq(r, http.MethodPatch, "/api/group/leave/not-uuid", auth, nil).Code)
	})

	t.Run("Error handling for group endpoints", func(t *testing.T) {
		errCourseSvc := &mockCourseService{
			isEnrolledOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) { return true, nil },
			isTeacherOrAdminFunc:  func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) { return true, nil },
		}
		errGroupSvc := &mockGroupService{
			listCourseGroupsFunc: func(cUuid uuid.UUID) ([]Group, error) { return nil, ErrCourseNotFound },
			getGroupFunc:         func(gUuid uuid.UUID) (*Group, error) { return nil, ErrGroupNotFound },
			createGroupFunc:      func(cUuid uuid.UUID, name string, maxMembers int) (*Group, error) { return nil, ErrCourseNotFound },
			bulkCreateGroupsFunc: func(cUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error) {
				return nil, errors.New("db error")
			},
			updateGroupFunc: func(gUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
				return nil, ErrGroupMaxMembersTooSmall
			},
			deleteGroupFunc:            func(gUuid uuid.UUID) error { return errors.New("db error") },
			userJoinGroupFunc:          func(gUuid, uUuid uuid.UUID) (*Group, error) { return nil, ErrUserAlreadyInGroup },
			removeStudentFromGroupFunc: func(gUuid, uUuid uuid.UUID) (*Group, error) { return nil, ErrUserNotInGroup },
		}

		errR, _ := setupCourseTestRouter(errCourseSvc, errGroupSvc)

		assert.Equal(t, http.StatusNotFound, doReq(errR, http.MethodGet, "/api/course/"+cID+"/group", auth, nil).Code)
		assert.Equal(t, http.StatusNotFound, doReq(errR, http.MethodGet, "/api/group/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusNotFound, doReq(errR, http.MethodPost, "/api/group", auth, CreateGroupDto{CourseUuid: cID, Name: "Group Valid", MaxMembers: 4}).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(errR, http.MethodPost, "/api/group/bulk", auth, BulkCreateGroupsDto{CourseUuid: cID, Count: 2, MaxMembers: 4}).Code)
		assert.Equal(t, http.StatusNotFound, doReq(errR, http.MethodPost, "/api/group/join/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusNotFound, doReq(errR, http.MethodPatch, "/api/group/leave/"+gID, auth, nil).Code)

		// Patch & Delete & Join & Leave domain errors when group exists
		patchDeleteGroupSvc := &mockGroupService{
			getGroupFunc: func(gUuid uuid.UUID) (*Group, error) {
				return &Group{Uuid: gUuid, Course: &Course{Uuid: courseUuid}}, nil
			},
			updateGroupFunc: func(gUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
				return nil, ErrGroupMaxMembersTooSmall
			},
			deleteGroupFunc: func(gUuid uuid.UUID) error {
				return errors.New("delete error")
			},
			userJoinGroupFunc: func(gUuid, uUuid uuid.UUID) (*Group, error) {
				return nil, ErrUserAlreadyInGroup
			},
			removeStudentFromGroupFunc: func(gUuid, uUuid uuid.UUID) (*Group, error) {
				return nil, ErrUserNotInGroup
			},
		}
		patchDeleteR, _ := setupCourseTestRouter(errCourseSvc, patchDeleteGroupSvc)
		assert.Equal(t, http.StatusConflict, doReq(patchDeleteR, http.MethodPost, "/api/group/join/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusBadRequest, doReq(patchDeleteR, http.MethodPatch, "/api/group/leave/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusBadRequest, doReq(patchDeleteR, http.MethodPatch, "/api/group/"+gID, auth, UpdateGroupDto{Name: "Renamed", MaxMembers: 4}).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(patchDeleteR, http.MethodDelete, "/api/group/"+gID, auth, nil).Code)

		// Authorization failures: unenrolled user cannot get, join, leave, or modify a group
		unauthCourseSvc := &mockCourseService{
			isEnrolledOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) { return false, nil },
			isTeacherOrAdminFunc:  func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) { return false, nil },
		}
		unauthR, _ := setupCourseTestRouter(unauthCourseSvc, patchDeleteGroupSvc)
		assert.Equal(t, http.StatusForbidden, doReq(unauthR, http.MethodGet, "/api/group/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusForbidden, doReq(unauthR, http.MethodPost, "/api/group/join/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusForbidden, doReq(unauthR, http.MethodPatch, "/api/group/leave/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusForbidden, doReq(unauthR, http.MethodPatch, "/api/group/"+gID, auth, UpdateGroupDto{Name: "Renamed", MaxMembers: 4}).Code)
		assert.Equal(t, http.StatusForbidden, doReq(unauthR, http.MethodDelete, "/api/group/"+gID, auth, nil).Code)

		// Missing group course association returns 500 without bypassing auth
		nilCourseGroupSvc := &mockGroupService{
			getGroupFunc: func(gUuid uuid.UUID) (*Group, error) {
				return &Group{Uuid: gUuid, Course: nil}, nil
			},
		}
		nilCourseR, _ := setupCourseTestRouter(errCourseSvc, nilCourseGroupSvc)
		assert.Equal(t, http.StatusInternalServerError, doReq(nilCourseR, http.MethodGet, "/api/group/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(nilCourseR, http.MethodPost, "/api/group/join/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(nilCourseR, http.MethodPatch, "/api/group/leave/"+gID, auth, nil).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(nilCourseR, http.MethodPatch, "/api/group/"+gID, auth, UpdateGroupDto{Name: "Renamed", MaxMembers: 4}).Code)
		assert.Equal(t, http.StatusInternalServerError, doReq(nilCourseR, http.MethodDelete, "/api/group/"+gID, auth, nil).Code)
	})
}

func TestCourseCtn_GradingScaleEndpoints(t *testing.T) {
	userUuid := uuid.New()
	auth := generateAuthHeader(t, userUuid, authutils.ROLE_USER)
	courseUuid := uuid.New()
	cID := courseUuid.String()

	mockSvc := &mockCourseService{
		isTeacherOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
			return true, nil
		},
		createCourseFunc: func(c *Course, creatorUuid uuid.UUID) (*Course, error) {
			if err := c.GradingScale.Validate(); err != nil {
				return nil, err
			}
			c.Uuid = courseUuid
			return c, nil
		},
		updateCourseFunc: func(cUuid uuid.UUID, updated *Course) (*Course, error) {
			if err := updated.GradingScale.Validate(); err != nil {
				return nil, err
			}
			updated.Uuid = cUuid
			return updated, nil
		},
	}

	router, _ := setupCourseTestRouter(mockSvc)

	t.Run("Create course with custom grading scale succeeds", func(t *testing.T) {
		payload := CreateCourseDto{
			Title: "Advanced Algorithms",
			Term:  "AUTUMN",
			Year:  2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 92,
				ScaleB: 82,
				ScaleC: 65,
				ScaleD: 52,
				ScaleE: 42,
			},
		}
		resp := doReq(router, http.MethodPost, "/api/course", auth, payload)
		assert.Equal(t, http.StatusCreated, resp.Code)
	})

	t.Run("Create course with invalid grading scale returns 400", func(t *testing.T) {
		payload := CreateCourseDto{
			Title: "Advanced Algorithms",
			Term:  "AUTUMN",
			Year:  2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 80,
				ScaleB: 85, // invalid
				ScaleC: 60,
				ScaleD: 50,
				ScaleE: 40,
			},
		}
		resp := doReq(router, http.MethodPost, "/api/course", auth, payload)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Update course with custom grading scale succeeds", func(t *testing.T) {
		payload := UpdateCourseDto{
			Title: "Advanced Algorithms",
			Term:  "AUTUMN",
			Year:  2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 95,
				ScaleB: 85,
				ScaleC: 70,
				ScaleD: 55,
				ScaleE: 45,
			},
		}
		resp := doReq(router, http.MethodPut, "/api/course/"+cID, auth, payload)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Update course with invalid grading scale returns 400", func(t *testing.T) {
		payload := UpdateCourseDto{
			Title: "Advanced Algorithms",
			Term:  "AUTUMN",
			Year:  2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 90,
				ScaleB: 80,
				ScaleC: 60,
				ScaleD: 50,
				ScaleE: 0, // invalid
			},
		}
		resp := doReq(router, http.MethodPut, "/api/course/"+cID, auth, payload)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}
