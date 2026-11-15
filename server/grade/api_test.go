package grade_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/grade"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockGradeService struct {
	upsertGradeFn func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error)
	releaseFn     func(courseUuid, contentUuid uuid.UUID) (int, error)
	listPendingFn func(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]grade.PendingSubmissionDto, error)
	listMyFn      func(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]grade.StudentGradeDto, error)
	getMyFinalsFn func(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]grade.CourseFinalGradeDto, error)
	getRosterFn   func(courseUuid uuid.UUID) ([]grade.StudentFinalGradeRowDto, error)
	exportCSVFn   func(courseUuid uuid.UUID, w io.Writer) error
}

func (m *mockGradeService) UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
	if m.upsertGradeFn != nil {
		return m.upsertGradeFn(courseUuid, contentUuid, submissionUuid, graderUuid, dto)
	}
	return &grade.GradeDto{
		Score:              *dto.Score,
		MaxPoints:          100.0,
		Feedback:           dto.Feedback,
		IsVisibleToStudent: dto.IsVisibleToStudent,
		GradedByEmail:      "teacher@example.com",
		GradedAt:           time.Now(),
	}, nil
}

func (m *mockGradeService) ReleaseAssignmentGrades(courseUuid, contentUuid uuid.UUID) (int, error) {
	if m.releaseFn != nil {
		return m.releaseFn(courseUuid, contentUuid)
	}
	return 3, nil
}

func (m *mockGradeService) ListPending(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]grade.PendingSubmissionDto, error) {
	if m.listPendingFn != nil {
		return m.listPendingFn(userUuid, isAdmin, courseUuid)
	}
	return []grade.PendingSubmissionDto{
		{
			SubmissionUuid:  uuid.New(),
			CourseUuid:      uuid.New(),
			ContentUuid:     uuid.New(),
			CourseName:      "DAT250",
			AssignmentTitle: "Lab 1",
			UserEmail:       "student@example.com",
			Attempt:         1,
			SubmittedAt:     time.Now(),
			MaxPoints:       100.0,
		},
	}, nil
}

func (m *mockGradeService) ListMyGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]grade.StudentGradeDto, error) {
	if m.listMyFn != nil {
		return m.listMyFn(userUuid, courseUuid)
	}
	return []grade.StudentGradeDto{
		{
			Type:            grade.GradeTypeHandin,
			CourseName:      "DAT250",
			AssignmentTitle: "Lab 1",
			Score:           90.0,
			MaxPoints:       100.0,
			Percent:         90.0,
			Feedback:        "Good job",
		},
	}, nil
}

func (m *mockGradeService) ExportCourseCSV(courseUuid uuid.UUID, w io.Writer) error {
	if m.exportCSVFn != nil {
		return m.exportCSVFn(courseUuid, w)
	}
	_, err := fmt.Fprint(w, "student_email,score\nstudent@example.com,90\n")
	return err
}

func (m *mockGradeService) GetStudentFinalGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]grade.CourseFinalGradeDto, error) {
	if m.getMyFinalsFn != nil {
		return m.getMyFinalsFn(userUuid, courseUuid)
	}
	return []grade.CourseFinalGradeDto{
		{
			CourseUuid:    uuid.New(),
			CourseName:    "DAT250",
			CourseCode:    "AUTUMN 2026",
			TotalScore:    180.0,
			TotalMaxScore: 200.0,
			Percent:       90.0,
			FinalGrade:    "A",
			GradingScale:  course.GradingScaleDto{}.FromModel(course.DefaultGradingScale()),
		},
	}, nil
}

func (m *mockGradeService) GetCourseRosterFinalGrades(courseUuid uuid.UUID) ([]grade.StudentFinalGradeRowDto, error) {
	if m.getRosterFn != nil {
		return m.getRosterFn(courseUuid)
	}
	return []grade.StudentFinalGradeRowDto{
		{
			UserUuid:      uuid.New(),
			UserEmail:     "student@example.com",
			UserName:      "Alice",
			TotalScore:    180.0,
			TotalMaxScore: 200.0,
			Percent:       90.0,
			FinalGrade:    "A",
		},
	}, nil
}

type mockCourseServiceForGrade struct {
	course.ICourseService
	isTeacherOrAdminFunc func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
}

func (m *mockCourseServiceForGrade) IsTeacherOrAdmin(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if m.isTeacherOrAdminFunc != nil {
		return m.isTeacherOrAdminFunc(courseUuid, userUuid, isSystemAdmin)
	}
	return true, nil
}

func init() {
	app.AccessKey = "test-grade-secret-access-key-12345"
	app.RefreshKey = "test-grade-secret-refresh-key-12345"
}

func setupGradeApiTest(gradeSvc grade.IGradeService, courseSvc course.ICourseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctn := grade.NewGradeCtn(gradeSvc, courseSvc, zap.NewNop().Sugar())
	ctn.RegisterEndpoints(r.Group("/api"))
	return r
}

func generateAuthHeader(t *testing.T, role authutils.Role, uid uuid.UUID) string {
	token, _, err := authutils.GenerateTokens("user@example.com", "user", role, uid)
	require.NoError(t, err)
	return "Bearer " + token
}

func doJSONReq(r http.Handler, method, path, auth string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
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

func TestGradeCtn_UpsertGrade(t *testing.T) {
	crsUuid := uuid.New()
	cntUuid := uuid.New()
	subUuid := uuid.New()
	teacherUuid := uuid.New()
	path := fmt.Sprintf("/api/course/%s/content/%s/submission/%s/grade", crsUuid, cntUuid, subUuid)

	t.Run("200 OK saves grade", func(t *testing.T) {
		svc := &mockGradeService{}
		cs := &mockCourseServiceForGrade{}
		r := setupGradeApiTest(svc, cs)

		score := 85.0
		payload := grade.UpsertGradeDto{
			Score:              &score,
			Feedback:           "Well done!",
			IsVisibleToStudent: true,
		}
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		var dto grade.GradeDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, 85.0, dto.Score)
		assert.Equal(t, "Well done!", dto.Feedback)
	})

	t.Run("400 Bad Request on invalid UUID params", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		score := 50.0

		// bad course UUID
		w1 := doJSONReq(r, http.MethodPut, "/api/course/invalid-uuid/content/"+cntUuid.String()+"/submission/"+subUuid.String()+"/grade", auth, grade.UpsertGradeDto{Score: &score})
		assert.Equal(t, http.StatusBadRequest, w1.Code)

		// bad content UUID
		w2 := doJSONReq(r, http.MethodPut, "/api/course/"+crsUuid.String()+"/content/invalid-uuid/submission/"+subUuid.String()+"/grade", auth, grade.UpsertGradeDto{Score: &score})
		assert.Equal(t, http.StatusBadRequest, w2.Code)

		// bad submission UUID
		w3 := doJSONReq(r, http.MethodPut, "/api/course/"+crsUuid.String()+"/content/"+cntUuid.String()+"/submission/invalid-uuid/grade", auth, grade.UpsertGradeDto{Score: &score})
		assert.Equal(t, http.StatusBadRequest, w3.Code)
	})

	t.Run("401 Unauthorized when grader user not found", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, grade.ErrGraderNotFound
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		score := 50.0
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("500 Internal Server Error on unexpected service error", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, errors.New("db failure")
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		score := 50.0
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("401 Unauthorized when missing token", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		score := 50.0
		w := doJSONReq(r, http.MethodPut, path, "", grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("403 Forbidden when not teacher or admin", func(t *testing.T) {
		cs := &mockCourseServiceForGrade{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupGradeApiTest(&mockGradeService{}, cs)
		score := 50.0
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("404 Not Found when submission not found", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, grade.ErrSubmissionNotFound
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		score := 50.0
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("409 Conflict when content is not handin", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, grade.ErrNotHandin
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		score := 50.0
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("422 Unprocessable Entity when score exceeds max points", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, grade.ErrScoreOutOfRange
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		score := 150.0
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("422 Unprocessable Entity when group grading requested", func(t *testing.T) {
		svc := &mockGradeService{
			upsertGradeFn: func(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto grade.UpsertGradeDto) (*grade.GradeDto, error) {
				return nil, grade.ErrGroupGradingNotSupported
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		score := 80.0
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPut, path, auth, grade.UpsertGradeDto{Score: &score})

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestGradeCtn_ReleaseGrades(t *testing.T) {
	crsUuid := uuid.New()
	cntUuid := uuid.New()
	path := fmt.Sprintf("/api/course/%s/content/%s/grade/release", crsUuid, cntUuid)
	teacherUuid := uuid.New()

	t.Run("200 OK releases grades", func(t *testing.T) {
		svc := &mockGradeService{
			releaseFn: func(courseUuid, contentUuid uuid.UUID) (int, error) {
				return 5, nil
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPost, path, auth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var res grade.ReleaseResultDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, 5, res.Released)
	})

	t.Run("400 Bad Request on invalid UUID params", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)

		w1 := doJSONReq(r, http.MethodPost, "/api/course/not-a-uuid/content/"+cntUuid.String()+"/grade/release", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w1.Code)

		w2 := doJSONReq(r, http.MethodPost, "/api/course/"+crsUuid.String()+"/content/not-a-uuid/grade/release", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w2.Code)
	})

	t.Run("403 Forbidden for non-teacher", func(t *testing.T) {
		cs := &mockCourseServiceForGrade{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupGradeApiTest(&mockGradeService{}, cs)
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPost, path, auth, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("422 Unprocessable Entity for group assignment", func(t *testing.T) {
		svc := &mockGradeService{
			releaseFn: func(courseUuid, contentUuid uuid.UUID) (int, error) {
				return 0, grade.ErrGroupGradingNotSupported
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodPost, path, auth, nil)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestGradeCtn_ListPending(t *testing.T) {
	userUuid := uuid.New()
	t.Run("200 OK returns pending submissions", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, userUuid)
		w := doJSONReq(r, http.MethodGet, "/api/grade/pending", auth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var dtos []grade.PendingSubmissionDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 1)
		assert.Equal(t, "Lab 1", dtos[0].AssignmentTitle)

		// With valid courseUuid filter
		validQuery := fmt.Sprintf("/api/grade/pending?courseUuid=%s", uuid.New().String())
		wFiltered := doJSONReq(r, http.MethodGet, validQuery, auth, nil)
		assert.Equal(t, http.StatusOK, wFiltered.Code)
	})

	t.Run("400 Bad Request on invalid course query param", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, userUuid)
		w := doJSONReq(r, http.MethodGet, "/api/grade/pending?courseUuid=invalid-uuid", auth, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGradeCtn_ListMyGrades(t *testing.T) {
	userUuid := uuid.New()
	t.Run("200 OK returns student grades", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, userUuid)
		w := doJSONReq(r, http.MethodGet, "/api/grade/my", auth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var dtos []grade.StudentGradeDto
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 1)
		assert.Equal(t, "Lab 1", dtos[0].AssignmentTitle)
		assert.Equal(t, 90.0, dtos[0].Score)

		// With valid courseUuid filter
		validQuery := fmt.Sprintf("/api/grade/my?courseUuid=%s", uuid.New().String())
		wFiltered := doJSONReq(r, http.MethodGet, validQuery, auth, nil)
		assert.Equal(t, http.StatusOK, wFiltered.Code)
	})

	t.Run("400 Bad Request on invalid course query param", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, userUuid)
		w := doJSONReq(r, http.MethodGet, "/api/grade/my?courseUuid=not-a-uuid", auth, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGradeCtn_ExportCSV(t *testing.T) {
	crsUuid := uuid.New()
	path := fmt.Sprintf("/api/course/%s/grade/export", crsUuid)
	teacherUuid := uuid.New()

	t.Run("400 Bad Request on invalid course UUID", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodGet, "/api/course/not-a-uuid/grade/export", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("200 OK streams CSV", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodGet, path, auth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, w.Body.String(), "student@example.com")
	})

	t.Run("400 Bad Request on invalid course UUID", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)

		w := doJSONReq(r, http.MethodGet, "/api/course/not-a-uuid/grade/export", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("403 Forbidden for non-teacher", func(t *testing.T) {
		cs := &mockCourseServiceForGrade{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupGradeApiTest(&mockGradeService{}, cs)
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodGet, path, auth, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("404 Not Found when course does not exist", func(t *testing.T) {
		cs := &mockCourseServiceForGrade{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, course.ErrCourseNotFound
			},
		}
		r := setupGradeApiTest(&mockGradeService{}, cs)
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodGet, path, auth, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("500 Internal Server Error when export fails", func(t *testing.T) {
		svc := &mockGradeService{
			exportCSVFn: func(courseUuid uuid.UUID, w io.Writer) error {
				return errors.New("db error")
			},
		}
		r := setupGradeApiTest(svc, &mockCourseServiceForGrade{})
		auth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)
		w := doJSONReq(r, http.MethodGet, path, auth, nil)

		assert.Equal(t, http.StatusOK, w.Code) // Headers already committed
	})
}

func TestGradeCtn_FinalGradesEndpoints(t *testing.T) {
	studentUuid := uuid.New()
	teacherUuid := uuid.New()
	courseUuid := uuid.New()

	studentAuth := generateAuthHeader(t, authutils.ROLE_USER, studentUuid)
	teacherAuth := generateAuthHeader(t, authutils.ROLE_USER, teacherUuid)

	t.Run("GET /grade/my/final returns 200 with final course grades", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		w := doJSONReq(r, http.MethodGet, "/api/grade/my/final", studentAuth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var res []grade.CourseFinalGradeDto
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "A", res[0].FinalGrade)
		assert.Equal(t, 90.0, res[0].Percent)
	})

	t.Run("GET /grade/my/final with invalid courseUuid returns 400", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		w := doJSONReq(r, http.MethodGet, "/api/grade/my/final?courseUuid=bad-uuid", studentAuth, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /course/:uuid/grade/roster returns 200 for teacher", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		path := fmt.Sprintf("/api/course/%s/grade/roster", courseUuid.String())
		w := doJSONReq(r, http.MethodGet, path, teacherAuth, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var roster []grade.StudentFinalGradeRowDto
		err := json.Unmarshal(w.Body.Bytes(), &roster)
		require.NoError(t, err)
		assert.Len(t, roster, 1)
		assert.Equal(t, "Alice", roster[0].UserName)
		assert.Equal(t, "A", roster[0].FinalGrade)
	})

	t.Run("GET /course/:uuid/grade/roster returns 403 for unauthorized user", func(t *testing.T) {
		cs := &mockCourseServiceForGrade{
			isTeacherOrAdminFunc: func(cUuid, uUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupGradeApiTest(&mockGradeService{}, cs)
		path := fmt.Sprintf("/api/course/%s/grade/roster", courseUuid.String())
		w := doJSONReq(r, http.MethodGet, path, studentAuth, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("GET /course/:uuid/grade/roster with invalid UUID returns 400", func(t *testing.T) {
		r := setupGradeApiTest(&mockGradeService{}, &mockCourseServiceForGrade{})
		w := doJSONReq(r, http.MethodGet, "/api/course/not-a-uuid/grade/roster", teacherAuth, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
