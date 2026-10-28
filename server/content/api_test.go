package content_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockContentService struct {
	uploadStaticPageFunc func(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*content.StaticPage, error)
	getStaticPageFunc    func(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (*content.StaticPage, error)
	listStaticPagesFunc  func(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]content.StaticPage, error)
	getFileStreamFunc    func(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *content.StaticPage, error)
	updateStaticPageFunc func(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*content.StaticPage, error)
	deleteStaticPageFunc func(courseUuid, moduleUuid, contentUuid uuid.UUID) error
}

func (m *mockContentService) UploadStaticPage(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*content.StaticPage, error) {
	if m.uploadStaticPageFunc != nil {
		return m.uploadStaticPageFunc(courseUuid, moduleUuid, title, fileName, fileSize, contentType, fileReader, description, sourceUrl)
	}
	return &content.StaticPage{
		Content: &content.Content{
			Uuid:     uuid.New(),
			Title:    title,
			Position: 1,
		},
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		SourceURL:   sourceUrl,
		RichText:    description,
	}, nil
}

func (m *mockContentService) GetStaticPage(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (*content.StaticPage, error) {
	if m.getStaticPageFunc != nil {
		return m.getStaticPageFunc(courseUuid, moduleUuid, contentUuid, userUuid, isPrivileged)
	}
	return &content.StaticPage{
		Content: &content.Content{
			Uuid:     contentUuid,
			Title:    "Sample Page",
			Position: 1,
		},
		FileName:    "sample.pdf",
		FileSize:    100,
		ContentType: "application/pdf",
	}, nil
}

func (m *mockContentService) ListStaticPages(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]content.StaticPage, error) {
	if m.listStaticPagesFunc != nil {
		return m.listStaticPagesFunc(courseUuid, moduleUuid, userUuid, isPrivileged)
	}
	return []content.StaticPage{
		{
			Content: &content.Content{
				Uuid:     uuid.New(),
				Title:    "Page 1",
				Position: 1,
			},
			FileName: "page1.pdf",
		},
	}, nil
}

func (m *mockContentService) GetFileStream(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *content.StaticPage, error) {
	if m.getFileStreamFunc != nil {
		return m.getFileStreamFunc(courseUuid, moduleUuid, contentUuid, userUuid, isPrivileged)
	}
	sp := &content.StaticPage{
		Content: &content.Content{
			Uuid:  contentUuid,
			Title: "Stream File",
		},
		FileName:    "file.pdf",
		FileSize:    12,
		ContentType: "application/pdf",
		StorageKey:  "key",
	}
	return io.NopCloser(strings.NewReader("pdf-content-bytes")), sp, nil
}

func (m *mockContentService) UpdateStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*content.StaticPage, error) {
	if m.updateStaticPageFunc != nil {
		return m.updateStaticPageFunc(courseUuid, moduleUuid, contentUuid, title, description)
	}
	return &content.StaticPage{
		Content: &content.Content{
			Uuid:  contentUuid,
			Title: title,
		},
		RichText: description,
	}, nil
}

func (m *mockContentService) DeleteStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID) error {
	if m.deleteStaticPageFunc != nil {
		return m.deleteStaticPageFunc(courseUuid, moduleUuid, contentUuid)
	}
	return nil
}

type mockCourseServiceForContent struct {
	course.ICourseService
	isTeacherOrAdminFunc  func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
	isEnrolledOrAdminFunc func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
}

func (m *mockCourseServiceForContent) IsTeacherOrAdmin(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if m.isTeacherOrAdminFunc != nil {
		return m.isTeacherOrAdminFunc(courseUuid, userUuid, isSystemAdmin)
	}
	return true, nil
}

func (m *mockCourseServiceForContent) IsEnrolledOrAdmin(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if m.isEnrolledOrAdminFunc != nil {
		return m.isEnrolledOrAdminFunc(courseUuid, userUuid, isSystemAdmin)
	}
	return true, nil
}

func init() {
	app.AccessKey = "test-content-secret-access-key-12345"
	app.RefreshKey = "test-content-secret-refresh-key-12345"
}

func setupContentApiTest(contentSvc content.IContentService, courseSvc course.ICourseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ctn := content.NewContentCtn(contentSvc, courseSvc, zap.NewNop().Sugar())
	ctn.RegisterEndpoints(r.Group("/api"))
	return r
}

func generateAuthHeader(t *testing.T, role authutils.Role) string {
	token, _, err := authutils.GenerateTokens("teacher@hvl.no", "teacher", role, uuid.New())
	require.NoError(t, err)
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

func doMultipartReq(r http.Handler, method, path, auth string, fields map[string]string, fileName string, fileBytes []byte) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	if fileName != "" && fileBytes != nil {
		part, _ := writer.CreateFormFile("file", fileName)
		_, _ = part.Write(fileBytes)
	}

	_ = writer.Close()

	req, _ := http.NewRequest(method, path, &buf)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestContentCtn_Upload(t *testing.T) {
	assert.NotNil(t, content.NewContentCtn(&mockContentService{}, &mockCourseServiceForContent{}, nil))

	courseUuid := uuid.New()
	moduleUuid := uuid.New()
	auth := generateAuthHeader(t, authutils.ROLE_USER)

	t.Run("upload file successfully", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		fields := map[string]string{
			"title":       "Syllabus PDF",
			"description": "Course syllabus document",
		}
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, fields, "syllabus.pdf", []byte("%PDF-1.4 test pdf content"))
		assert.Equal(t, http.StatusCreated, w.Code)

		var res content.StaticPageDto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, "Syllabus PDF", res.Title)
		assert.Equal(t, "syllabus.pdf", res.FileName)
	})

	t.Run("upload link/text without file", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		fields := map[string]string{
			"title":     "Web Link",
			"sourceUrl": "https://example.com",
		}
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, fields, "", nil)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("upload disallowed file type returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Bad HTML"}, "test.html", []byte("<script>alert(1)</script>"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload javascript sourceUrl returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		fields := map[string]string{
			"title":     "Bad Link",
			"sourceUrl": "javascript:alert(1)",
		}
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, fields, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload title too long returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		fields := map[string]string{
			"title":     strings.Repeat("a", 256),
			"sourceUrl": "https://example.com",
		}
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, fields, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload title too short returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		fields := map[string]string{
			"title":     "a",
			"sourceUrl": "https://example.com",
		}
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, fields, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload unsupported extension returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "File"}, "test.unknown", []byte("xyz"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload PDF with mismatching content returns bad request", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Fake PDF"}, "test.pdf", []byte("plain text instead of pdf signature"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing all required fields", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{}, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUIDs", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/invalid/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/invalid/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("forbidden when not teacher or admin", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("course not found during role check", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, course.ErrCourseNotFound
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service not found error", func(t *testing.T) {
		contentMock := &mockContentService{
			uploadStaticPageFunc: func(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*content.StaticPage, error) {
				return nil, content.ErrModuleNotFound
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unexpected service error", func(t *testing.T) {
		contentMock := &mockContentService{
			uploadStaticPageFunc: func(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*content.StaticPage, error) {
				return nil, errors.New("db explosion")
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doMultipartReq(r, http.MethodPost, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, map[string]string{"title": "Test"}, "", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestContentCtn_ListAndGet(t *testing.T) {
	courseUuid := uuid.New()
	moduleUuid := uuid.New()
	pageUuid := uuid.New()
	auth := generateAuthHeader(t, authutils.ROLE_USER)

	t.Run("list static pages successfully", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var list []content.StaticPageDto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
		assert.Len(t, list, 1)
	})

	t.Run("list with invalid course UUID", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/not-a-uuid/module/"+moduleUuid.String()+"/static-page", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("list with invalid module UUID", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/not-a-uuid/static-page", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("list module not found", func(t *testing.T) {
		contentMock := &mockContentService{
			listStaticPagesFunc: func(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]content.StaticPage, error) {
				return nil, content.ErrModuleNotFound
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("list module locked returns 403", func(t *testing.T) {
		contentMock := &mockContentService{
			listStaticPagesFunc: func(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]content.StaticPage, error) {
				return nil, content.ErrModuleLocked
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("list forbidden when not enrolled", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isEnrolledOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page", auth, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("get static page forbidden when not enrolled", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isEnrolledOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("get static page successfully", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get static page invalid page UUID", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/not-uuid", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodGet, "/api/course/not-uuid/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/not-uuid/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get static page not found", func(t *testing.T) {
		contentMock := &mockContentService{
			getStaticPageFunc: func(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (*content.StaticPage, error) {
				return nil, content.ErrContentNotFound
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestContentCtn_Preview(t *testing.T) {
	courseUuid := uuid.New()
	moduleUuid := uuid.New()
	pageUuid := uuid.New()
	auth := generateAuthHeader(t, authutils.ROLE_USER)

	t.Run("preview file successfully with inline header", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Disposition"), "inline")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "filename=")
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox")
		assert.Equal(t, "pdf-content-bytes", w.Body.String())
	})

	t.Run("preview object not found in storage", func(t *testing.T) {
		contentMock := &mockContentService{
			getFileStreamFunc: func(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *content.StaticPage, error) {
				return nil, nil, storage.ErrObjectNotFound
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("preview invalid page UUID", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/invalid-uuid/preview", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodGet, "/api/course/invalid-uuid/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/invalid-uuid/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("preview file with empty content type defaults to octet-stream", func(t *testing.T) {
		contentMock := &mockContentService{
			getFileStreamFunc: func(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *content.StaticPage, error) {
				return io.NopCloser(strings.NewReader("bytes")), &content.StaticPage{
					Content:     &content.Content{Uuid: contentUuid, Title: "No MIME"},
					FileName:    "file.bin",
					ContentType: "",
				}, nil
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	})

	t.Run("preview forbidden when not enrolled", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isEnrolledOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doReq(r, http.MethodGet, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String()+"/preview", auth, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestContentCtn_Update(t *testing.T) {
	courseUuid := uuid.New()
	moduleUuid := uuid.New()
	pageUuid := uuid.New()
	auth := generateAuthHeader(t, authutils.ROLE_USER)

	t.Run("update static page metadata successfully", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		body := content.UpdateStaticPageDto{
			Title:       "Updated Title",
			Description: "Updated Description",
		}
		w := doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, body)
		assert.Equal(t, http.StatusOK, w.Code)

		var res content.StaticPageDto
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, "Updated Title", res.Title)
		assert.Equal(t, "Updated Description", res.Description)
	})

	t.Run("update invalid request body", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, "{invalid-json")
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodPut, "/api/course/not-uuid/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, content.UpdateStaticPageDto{Title: "T"})
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/not-uuid/static-page/"+pageUuid.String(), auth, content.UpdateStaticPageDto{Title: "T"})
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/not-uuid", auth, content.UpdateStaticPageDto{Title: "T"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update forbidden", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		body := content.UpdateStaticPageDto{Title: "Updated"}
		w := doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, body)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("update static page service error", func(t *testing.T) {
		contentMock := &mockContentService{
			updateStaticPageFunc: func(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*content.StaticPage, error) {
				return nil, errors.New("update failed")
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		body := content.UpdateStaticPageDto{Title: "Updated"}
		w := doReq(r, http.MethodPut, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, body)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestContentCtn_Delete(t *testing.T) {
	courseUuid := uuid.New()
	moduleUuid := uuid.New()
	pageUuid := uuid.New()
	auth := generateAuthHeader(t, authutils.ROLE_USER)

	t.Run("delete static page successfully", func(t *testing.T) {
		r := setupContentApiTest(&mockContentService{}, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("delete static page not found", func(t *testing.T) {
		contentMock := &mockContentService{
			deleteStaticPageFunc: func(courseUuid, moduleUuid, contentUuid uuid.UUID) error {
				return content.ErrContentNotFound
			},
		}
		r := setupContentApiTest(contentMock, &mockCourseServiceForContent{})
		w := doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)

		w = doReq(r, http.MethodDelete, "/api/course/not-uuid/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/not-uuid/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/not-uuid", auth, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete forbidden", func(t *testing.T) {
		courseMock := &mockCourseServiceForContent{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupContentApiTest(&mockContentService{}, courseMock)
		w := doReq(r, http.MethodDelete, "/api/course/"+courseUuid.String()+"/module/"+moduleUuid.String()+"/static-page/"+pageUuid.String(), auth, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
