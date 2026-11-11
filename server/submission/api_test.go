package submission_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockSubmissionService struct {
	resolveTargetFunc                   func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error)
	initFileSubmissionFunc              func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.UploadTicket, error)
	validateAndExtendTicketFunc         func(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*submission.UploadTicket, error)
	consumeUploadTicketFunc             func(ticketUuid uuid.UUID) error
	saveFromMessageFunc                 func(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error)
	notifySubmittedFunc                 func(target *submission.SubmissionTarget)
	notifySubmissionFailedFunc          func(userUuid, courseUuid uuid.UUID, contentTitle string)
	checkDeadlineRemindersFunc          func(now time.Time) (int, error)
	listMySubmissionsFunc               func(courseUuid, contentUuid, userUuid uuid.UUID) ([]submission.SubmissionDto, error)
	listCourseAssignmentSubmissionsFunc func(courseUuid, contentUuid uuid.UUID) ([]submission.SubmissionDto, error)
	getSubmissionForDownloadFunc        func(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error)
}

func (m *mockSubmissionService) ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
	if m.resolveTargetFunc != nil {
		return m.resolveTargetFunc(courseUuid, contentUuid, userUuid, submittedAt)
	}
	return &submission.SubmissionTarget{ContentID: 1, ContentTitle: "Assignment", CourseUuid: courseUuid, UserID: 1, UserUuid: userUuid, UserEmail: "student@hvl.no"}, nil
}

func (m *mockSubmissionService) InitFileSubmission(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.UploadTicket, error) {
	if m.initFileSubmissionFunc != nil {
		return m.initFileSubmissionFunc(courseUuid, contentUuid, userUuid, submittedAt)
	}
	return &submission.UploadTicket{
		Uuid:        uuid.New(),
		CourseID:    1,
		ContentID:   1,
		UserID:      1,
		UserEmail:   "student@hvl.no",
		SubmittedAt: submittedAt,
		ExpiresAt:   submittedAt.Add(30 * time.Minute),
	}, nil
}

func (m *mockSubmissionService) ValidateAndExtendTicket(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*submission.UploadTicket, error) {
	if m.validateAndExtendTicketFunc != nil {
		return m.validateAndExtendTicketFunc(ticketUuid, courseUuid, contentUuid, userUuid, now)
	}
	return &submission.UploadTicket{
		Uuid:         ticketUuid,
		CourseID:     1,
		ContentID:    1,
		ContentTitle: "Lab 1",
		UserID:       1,
		UserEmail:    "student@hvl.no",
		SubmittedAt:  now.Add(-5 * time.Minute),
		ExpiresAt:    now.Add(25 * time.Minute),
	}, nil
}

func (m *mockSubmissionService) ConsumeUploadTicket(ticketUuid uuid.UUID) error {
	if m.consumeUploadTicketFunc != nil {
		return m.consumeUploadTicketFunc(ticketUuid)
	}
	return nil
}

func (m *mockSubmissionService) ListMySubmissions(courseUuid, contentUuid, userUuid uuid.UUID) ([]submission.SubmissionDto, error) {
	if m.listMySubmissionsFunc != nil {
		return m.listMySubmissionsFunc(courseUuid, contentUuid, userUuid)
	}
	return []submission.SubmissionDto{}, nil
}

func (m *mockSubmissionService) ListCourseAssignmentSubmissions(courseUuid, contentUuid uuid.UUID) ([]submission.SubmissionDto, error) {
	if m.listCourseAssignmentSubmissionsFunc != nil {
		return m.listCourseAssignmentSubmissionsFunc(courseUuid, contentUuid)
	}
	return []submission.SubmissionDto{}, nil
}

func (m *mockSubmissionService) GetSubmissionForDownload(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error) {
	if m.getSubmissionForDownloadFunc != nil {
		return m.getSubmissionForDownloadFunc(courseUuid, contentUuid, subUuid, userUuid, isPrivileged)
	}
	uID := uint(1)
	return &submission.IndividualSubmission{
		Submission: submission.Submission{
			Uuid:        subUuid,
			PayloadType: submission.PayloadTypeFile,
			FileName:    "file.pdf",
			FileSize:    100,
			ContentType: "application/pdf",
			StorageKey:  "submissions/key.pdf",
		},
		UserID: &uID,
	}, nil
}

func (m *mockSubmissionService) SaveFromMessage(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error) {
	if m.saveFromMessageFunc != nil {
		return m.saveFromMessageFunc(msg)
	}
	return &submission.IndividualSubmission{
		Submission: submission.Submission{
			Uuid:        msg.Uuid,
			ContentID:   msg.ContentID,
			Attempt:     1,
			IsLate:      false,
			PayloadType: msg.PayloadType,
			PayloadURL:  msg.PayloadURL,
			FileName:    msg.FileName,
			FileSize:    msg.FileSize,
			ContentType: msg.ContentType,
			StorageKey:  msg.StorageKey,
			SubmittedAt: msg.SubmittedAt,
		},
		UserID:    &msg.UserID,
		UserEmail: msg.UserEmail,
	}, nil
}

func (m *mockSubmissionService) NotifySubmitted(target *submission.SubmissionTarget) {
	if m.notifySubmittedFunc != nil {
		m.notifySubmittedFunc(target)
	}
}

func (m *mockSubmissionService) NotifySubmissionFailed(userUuid, courseUuid uuid.UUID, contentTitle string) {
	if m.notifySubmissionFailedFunc != nil {
		m.notifySubmissionFailedFunc(userUuid, courseUuid, contentTitle)
	}
}

func (m *mockSubmissionService) CheckDeadlineReminders(now time.Time) (int, error) {
	if m.checkDeadlineRemindersFunc != nil {
		return m.checkDeadlineRemindersFunc(now)
	}
	return 0, nil
}

type mockPublisher struct {
	publishFunc func(ctx context.Context, msg submission.SubmissionMessage) error
	lastMsg     *submission.SubmissionMessage
}

func (m *mockPublisher) Publish(ctx context.Context, msg submission.SubmissionMessage) error {
	m.lastMsg = &msg
	if m.publishFunc != nil {
		return m.publishFunc(ctx, msg)
	}
	return nil
}

type mockCourseServiceForSubmission struct {
	course.ICourseService
	isTeacherOrAdminFunc func(courseUuid, userUuid uuid.UUID, isAdmin bool) (bool, error)
}

func (m *mockCourseServiceForSubmission) IsTeacherOrAdmin(courseUuid, userUuid uuid.UUID, isAdmin bool) (bool, error) {
	if m.isTeacherOrAdminFunc != nil {
		return m.isTeacherOrAdminFunc(courseUuid, userUuid, isAdmin)
	}
	return false, nil
}

type mockStorageService struct {
	uploadFunc func(key string, r io.Reader, contentType string) error
	getFunc    func(key string) (io.ReadCloser, string, error)
	deleteFunc func(key string) error
}

func (m *mockStorageService) Upload(key string, r io.Reader, contentType string) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(key, r, contentType)
	}
	return nil
}

func (m *mockStorageService) Get(key string) (io.ReadCloser, string, error) {
	if m.getFunc != nil {
		return m.getFunc(key)
	}
	return io.NopCloser(strings.NewReader("")), "", nil
}

func (m *mockStorageService) Delete(key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(key)
	}
	return nil
}

func setupTestRouter(svc submission.ISubmissionService, pub submission.ISubmissionPublisher) *gin.Engine {
	return setupTestRouterWithStorage(svc, pub, storage.NewMemoryStorageService(), &mockCourseServiceForSubmission{})
}

func setupTestRouterWithStorage(svc submission.ISubmissionService, pub submission.ISubmissionPublisher, st storage.IStorageService, cs course.ICourseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	ctn := submission.NewSubmissionCtn(svc, pub, st, cs, zap.NewNop().Sugar())
	ctn.RegisterEndpoints(api)
	return r
}

func genAuthHeader(t *testing.T, userUuid uuid.UUID) string {
	token, _, err := authutils.GenerateTokens("student@hvl.no", "student", authutils.ROLE_USER, userUuid)
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
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestSubmissionCtn_Submit(t *testing.T) {
	studentUuid := uuid.New()
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	auth := genAuthHeader(t, studentUuid)

	path := fmt.Sprintf("/api/course/%s/content/%s/submission", courseUuid, contentUuid)

	validPayload := submission.CreateSubmissionDto{
		PayloadType: submission.PayloadTypeURL,
		PayloadURL:  "https://git.app.uib.no/student/task1.git",
	}

	t.Run("202 Accepted on valid submission and enqueues message", func(t *testing.T) {
		notifiedSubmitted := false
		mockSvc := &mockSubmissionService{
			notifySubmittedFunc: func(target *submission.SubmissionTarget) {
				notifiedSubmitted = true
			},
		}
		mockPub := &mockPublisher{}
		r := setupTestRouter(mockSvc, mockPub)

		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.True(t, notifiedSubmitted)

		var resp submission.SubmissionAcceptedDto
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEqual(t, uuid.Nil, resp.Uuid)
		assert.Equal(t, "QUEUED", resp.Status)
		assert.False(t, resp.SubmittedAt.IsZero())

		assert.NotNil(t, mockPub.lastMsg)
		assert.Equal(t, resp.Uuid, mockPub.lastMsg.Uuid)
		assert.Equal(t, uint(1), mockPub.lastMsg.ContentID)
		assert.Equal(t, uint(1), mockPub.lastMsg.UserID)
		assert.Equal(t, studentUuid, mockPub.lastMsg.UserUuid)
		assert.Equal(t, courseUuid, mockPub.lastMsg.CourseUuid)
		assert.Equal(t, "Assignment", mockPub.lastMsg.ContentTitle)
		assert.Equal(t, "https://git.app.uib.no/student/task1.git", mockPub.lastMsg.PayloadURL)
	})

	t.Run("401 Unauthorized when missing token", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, "", validPayload)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("400 Bad Request on invalid UUID route param", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badCoursePath := fmt.Sprintf("/api/course/not-a-uuid/content/%s/submission", contentUuid)
		recCourse := doReq(r, http.MethodPost, badCoursePath, auth, validPayload)
		assert.Equal(t, http.StatusBadRequest, recCourse.Code)

		badPath := fmt.Sprintf("/api/course/%s/content/not-a-uuid/submission", courseUuid)
		rec := doReq(r, http.MethodPost, badPath, auth, validPayload)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request on invalid payload URL", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, map[string]string{
			"payloadType": "URL",
			"payloadUrl":  "not-a-valid-url",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		recJS := doReq(r, http.MethodPost, path, auth, map[string]string{
			"payloadType": "URL",
			"payloadUrl":  "javascript:alert(1)",
		})
		assert.Equal(t, http.StatusBadRequest, recJS.Code)

		recFile := doReq(r, http.MethodPost, path, auth, map[string]string{
			"payloadType": "URL",
			"payloadUrl":  "file:///etc/passwd",
		})
		assert.Equal(t, http.StatusBadRequest, recFile.Code)
	})

	t.Run("400 Bad Request on unsupported FILE payloadType", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, submission.CreateSubmissionDto{
			PayloadType: submission.PayloadTypeFile,
			PayloadURL:  "https://example.com/file.pdf",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "not supported yet")
	})

	t.Run("404 Not Found when content not found", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrContentNotFound
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 Bad Request when content is not an assignment", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrNotAssignment
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request when assignment type is unsupported", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrAssignmentTypeUnsupported
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request when deadline passed and late submissions rejected", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrDeadlinePassed
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("403 Forbidden when user is not enrolled as student", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrNotEnrolled
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("403 Forbidden when prerequisite module is locked", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, submission.ErrModuleLocked
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "prerequisite")
	})

	t.Run("500 Internal Server Error on unexpected service error", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			resolveTargetFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
				return nil, errors.New("unexpected db error")
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("503 Service Unavailable when publisher fails", func(t *testing.T) {
		notifiedFailed := false
		mockSvc := &mockSubmissionService{
			notifySubmissionFailedFunc: func(userUuid, courseUuid uuid.UUID, contentTitle string) {
				notifiedFailed = true
			},
		}
		mockPub := &mockPublisher{
			publishFunc: func(ctx context.Context, msg submission.SubmissionMessage) error {
				return errors.New("rabbitmq connection lost")
			},
		}
		r := setupTestRouter(mockSvc, mockPub)
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.True(t, notifiedFailed)
	})
}

func TestSubmissionCtn_InitFileSubmission(t *testing.T) {
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	userUuid := uuid.New()
	auth := genAuthHeader(t, userUuid)
	path := fmt.Sprintf("/api/course/%s/content/%s/submission/file/init", courseUuid, contentUuid)

	t.Run("201 Created on valid init", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, nil)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var dto submission.UploadTicketDto
		err := json.Unmarshal(rec.Body.Bytes(), &dto)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, dto.TicketUuid)
		assert.False(t, dto.SubmittedAt.IsZero())
	})

	t.Run("401 Unauthorized when missing token", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("400 Bad Request on invalid uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPath := fmt.Sprintf("/api/course/bad-uuid/content/%s/submission/file/init", contentUuid)
		rec := doReq(r, http.MethodPost, badPath, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("404 Not Found when content not found", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			initFileSubmissionFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.UploadTicket, error) {
				return nil, submission.ErrContentNotFound
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 Bad Request on invalid content uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPath := fmt.Sprintf("/api/course/%s/content/bad-uuid/submission/file/init", courseUuid)
		rec := doReq(r, http.MethodPost, badPath, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("500 Internal Server Error on service failure", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			initFileSubmissionFunc: func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.UploadTicket, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, auth, nil)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func createMultipartFileRequest(url, fieldName, fileName string, fileContent []byte, auth string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if fieldName != "" {
		part, err := writer.CreateFormFile(fieldName, fileName)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(fileContent); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return req, nil
}

func TestSubmissionCtn_UploadFileSubmission(t *testing.T) {
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	userUuid := uuid.New()
	ticketUuid := uuid.New()
	auth := genAuthHeader(t, userUuid)
	url := fmt.Sprintf("/api/course/%s/content/%s/submission/file/upload?ticket=%s", courseUuid, contentUuid, ticketUuid)

	t.Run("202 Accepted on valid file upload", func(t *testing.T) {
		memStorage := storage.NewMemoryStorageService()
		pub := &mockPublisher{}
		var notified *submission.SubmissionTarget
		r := setupTestRouterWithStorage(&mockSubmissionService{notifySubmittedFunc: func(target *submission.SubmissionTarget) {
			notified = target
		}}, pub, memStorage, &mockCourseServiceForSubmission{})

		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("%PDF-fake-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.NotNil(t, pub.lastMsg)
		assert.Equal(t, submission.PayloadTypeFile, pub.lastMsg.PayloadType)
		assert.Equal(t, "project.pdf", pub.lastMsg.FileName)
		assert.Equal(t, userUuid, pub.lastMsg.UserUuid)
		assert.Equal(t, courseUuid, pub.lastMsg.CourseUuid)
		assert.Equal(t, "Lab 1", pub.lastMsg.ContentTitle)
		assert.NotNil(t, notified)
		assert.Equal(t, userUuid, notified.UserUuid)
		assert.Equal(t, courseUuid, notified.CourseUuid)
		assert.Equal(t, "Lab 1", notified.ContentTitle)
	})

	t.Run("400 Bad Request on invalid ticket uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badUrl := fmt.Sprintf("/api/course/%s/content/%s/submission/file/upload?ticket=not-a-uuid", courseUuid, contentUuid)
		req, err := createMultipartFileRequest(badUrl, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request when ticket is expired", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			validateAndExtendTicketFunc: func(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*submission.UploadTicket, error) {
				return nil, submission.ErrTicketExpired
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request on unsupported file extension", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		req, err := createMultipartFileRequest(url, "file", "exploit.exe", []byte("binary"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Unsupported file extension")
	})

	t.Run("400 Bad Request when form has no file field", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		req, err := createMultipartFileRequest(url, "other", "project.pdf", []byte("pdf"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No file provided")
	})

	t.Run("400 Bad Request when missing ticket param", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		noTicketUrl := fmt.Sprintf("/api/course/%s/content/%s/submission/file/upload", courseUuid, contentUuid)
		req, err := createMultipartFileRequest(noTicketUrl, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid upload ticket")
	})

	t.Run("400 Bad Request on invalid path uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPathUrl := fmt.Sprintf("/api/course/bad-uuid/content/%s/submission/file/upload?ticket=%s", contentUuid, ticketUuid)
		req, err := createMultipartFileRequest(badPathUrl, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("400 Bad Request when ticket is already consumed", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			validateAndExtendTicketFunc: func(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*submission.UploadTicket, error) {
				return nil, submission.ErrTicketConsumed
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("500 Internal Server Error when validate ticket fails unexpectedly", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			validateAndExtendTicketFunc: func(ticketUuid, courseUuid, contentUuid, userUuid uuid.UUID, now time.Time) (*submission.UploadTicket, error) {
				return nil, errors.New("db connection failure")
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("500 Internal Server Error when storage upload fails", func(t *testing.T) {
		mockSt := &mockStorageService{
			uploadFunc: func(key string, r io.Reader, contentType string) error {
				return errors.New("storage disk full")
			},
		}
		r := setupTestRouterWithStorage(&mockSubmissionService{}, &mockPublisher{}, mockSt, &mockCourseServiceForSubmission{})
		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("%PDF-content"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("503 Service Unavailable when publisher fails during upload", func(t *testing.T) {
		notifiedFailed := false
		pub := &mockPublisher{
			publishFunc: func(ctx context.Context, msg submission.SubmissionMessage) error {
				return errors.New("rabbitmq error")
			},
		}
		r := setupTestRouterWithStorage(&mockSubmissionService{notifySubmissionFailedFunc: func(u, c uuid.UUID, title string) {
			notifiedFailed = true
			assert.Equal(t, userUuid, u)
			assert.Equal(t, courseUuid, c)
			assert.Equal(t, "Lab 1", title)
		}}, pub, storage.NewMemoryStorageService(), &mockCourseServiceForSubmission{})
		req, err := createMultipartFileRequest(url, "file", "project.pdf", []byte("pdf"), auth)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.True(t, notifiedFailed)
	})
}

func TestSubmissionCtn_ListMySubmissions(t *testing.T) {
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	userUuid := uuid.New()
	auth := genAuthHeader(t, userUuid)
	path := fmt.Sprintf("/api/course/%s/content/%s/submission/my", courseUuid, contentUuid)

	t.Run("200 OK returns submissions", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			listMySubmissionsFunc: func(courseUuid, contentUuid, userUuid uuid.UUID) ([]submission.SubmissionDto, error) {
				return []submission.SubmissionDto{
					{Uuid: uuid.New(), Attempt: 1, PayloadType: submission.PayloadTypeFile, FileName: "test.pdf"},
				}, nil
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusOK, rec.Code)

		var dtos []submission.SubmissionDto
		err := json.Unmarshal(rec.Body.Bytes(), &dtos)
		assert.NoError(t, err)
		assert.Len(t, dtos, 1)
	})

	t.Run("400 Bad Request on invalid uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPath := fmt.Sprintf("/api/course/bad-uuid/content/%s/submission/my", contentUuid)
		rec := doReq(r, http.MethodGet, badPath, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("500 Internal Server Error when service fails", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			listMySubmissionsFunc: func(courseUuid, contentUuid, userUuid uuid.UUID) ([]submission.SubmissionDto, error) {
				return nil, errors.New("db failure")
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestSubmissionCtn_ListCourseSubmissions(t *testing.T) {
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	userUuid := uuid.New()
	auth := genAuthHeader(t, userUuid)
	path := fmt.Sprintf("/api/course/%s/content/%s/submission", courseUuid, contentUuid)

	t.Run("400 Bad Request on invalid uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPath := fmt.Sprintf("/api/course/bad-uuid/content/%s/submission", contentUuid)
		rec := doReq(r, http.MethodGet, badPath, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("403 Forbidden when not teacher or admin", func(t *testing.T) {
		mockCs := &mockCourseServiceForSubmission{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isAdmin bool) (bool, error) {
				return false, nil
			},
		}
		r := setupTestRouterWithStorage(&mockSubmissionService{}, &mockPublisher{}, storage.NewMemoryStorageService(), mockCs)
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("200 OK when teacher", func(t *testing.T) {
		mockCs := &mockCourseServiceForSubmission{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isAdmin bool) (bool, error) {
				return true, nil
			},
		}
		mockSvc := &mockSubmissionService{
			listCourseAssignmentSubmissionsFunc: func(courseUuid, contentUuid uuid.UUID) ([]submission.SubmissionDto, error) {
				return []submission.SubmissionDto{
					{Uuid: uuid.New(), Attempt: 1, UserEmail: "student@hvl.no"},
				}, nil
			},
		}
		r := setupTestRouterWithStorage(mockSvc, &mockPublisher{}, storage.NewMemoryStorageService(), mockCs)
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("500 Internal Server Error when service fails", func(t *testing.T) {
		mockCs := &mockCourseServiceForSubmission{
			isTeacherOrAdminFunc: func(courseUuid, userUuid uuid.UUID, isAdmin bool) (bool, error) {
				return true, nil
			},
		}
		mockSvc := &mockSubmissionService{
			listCourseAssignmentSubmissionsFunc: func(courseUuid, contentUuid uuid.UUID) ([]submission.SubmissionDto, error) {
				return nil, errors.New("db failure")
			},
		}
		r := setupTestRouterWithStorage(mockSvc, &mockPublisher{}, storage.NewMemoryStorageService(), mockCs)
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestSubmissionCtn_DownloadSubmissionFile(t *testing.T) {
	courseUuid := uuid.New()
	contentUuid := uuid.New()
	subUuid := uuid.New()
	userUuid := uuid.New()
	auth := genAuthHeader(t, userUuid)
	path := fmt.Sprintf("/api/course/%s/content/%s/submission/%s/file", courseUuid, contentUuid, subUuid)

	t.Run("400 Bad Request on invalid uuid", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		badPath := fmt.Sprintf("/api/course/bad-uuid/content/%s/submission/%s/file", contentUuid, subUuid)
		rec := doReq(r, http.MethodGet, badPath, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("200 OK downloads file when authorized", func(t *testing.T) {
		memStorage := storage.NewMemoryStorageService()
		_ = memStorage.Upload("submissions/key.pdf", strings.NewReader("sample file content"), "application/pdf")

		r := setupTestRouterWithStorage(&mockSubmissionService{}, &mockPublisher{}, memStorage, &mockCourseServiceForSubmission{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "sample file content", rec.Body.String())
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	})

	t.Run("404 Not Found when submission not found", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			getSubmissionForDownloadFunc: func(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error) {
				return nil, submission.ErrSubmissionNotFound
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 Bad Request when submission is not a file payload", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			getSubmissionForDownloadFunc: func(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error) {
				uID := uint(1)
				return &submission.IndividualSubmission{
					Submission: submission.Submission{
						Uuid:        subUuid,
						PayloadType: submission.PayloadTypeURL,
					},
					UserID: &uID,
				}, nil
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "not a stored file")
	})

	t.Run("500 Internal Server Error when storage get fails", func(t *testing.T) {
		mockSt := &mockStorageService{
			getFunc: func(key string) (io.ReadCloser, string, error) {
				return nil, "", errors.New("storage read failure")
			},
		}
		r := setupTestRouterWithStorage(&mockSubmissionService{}, &mockPublisher{}, mockSt, &mockCourseServiceForSubmission{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("403 Forbidden when unauthorized", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			getSubmissionForDownloadFunc: func(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error) {
				return nil, submission.ErrUnauthorizedSubmission
			},
		}
		r := setupTestRouter(mockSvc, &mockPublisher{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("200 OK downloads file with default fallback name and content type", func(t *testing.T) {
		memStorage := storage.NewMemoryStorageService()
		_ = memStorage.Upload("submissions/fallback.bin", strings.NewReader("fallback content"), "")

		mockSvc := &mockSubmissionService{
			getSubmissionForDownloadFunc: func(courseUuid, contentUuid, subUuid, userUuid uuid.UUID, isPrivileged bool) (*submission.IndividualSubmission, error) {
				uID := uint(1)
				return &submission.IndividualSubmission{
					Submission: submission.Submission{
						Uuid:        subUuid,
						PayloadType: submission.PayloadTypeFile,
						FileName:    "",
						FileSize:    16,
						ContentType: "",
						StorageKey:  "submissions/fallback.bin",
					},
					UserID: &uID,
				}, nil
			},
		}

		r := setupTestRouterWithStorage(mockSvc, &mockPublisher{}, memStorage, &mockCourseServiceForSubmission{})
		rec := doReq(r, http.MethodGet, path, auth, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "fallback content", rec.Body.String())
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "submission_file")
	})
}
