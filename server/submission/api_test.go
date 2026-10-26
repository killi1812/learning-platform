package submission_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockSubmissionService struct {
	resolveTargetFunc   func(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error)
	saveFromMessageFunc func(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error)
}

func (m *mockSubmissionService) ResolveTarget(courseUuid, contentUuid, userUuid uuid.UUID, submittedAt time.Time) (*submission.SubmissionTarget, error) {
	if m.resolveTargetFunc != nil {
		return m.resolveTargetFunc(courseUuid, contentUuid, userUuid, submittedAt)
	}
	return &submission.SubmissionTarget{ContentID: 1, UserID: 1, UserEmail: "student@hvl.no"}, nil
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
			SubmittedAt: msg.SubmittedAt,
		},
		UserID:    &msg.UserID,
		UserEmail: msg.UserEmail,
	}, nil
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

func setupTestRouter(svc submission.ISubmissionService, pub submission.ISubmissionPublisher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	ctn := submission.NewSubmissionCtn(svc, pub, zap.NewNop().Sugar())
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
		mockSvc := &mockSubmissionService{}
		mockPub := &mockPublisher{}
		r := setupTestRouter(mockSvc, mockPub)

		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusAccepted, rec.Code)

		var resp submission.SubmissionAcceptedDto
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEqual(t, uuid.Nil, resp.Uuid)
		assert.Equal(t, "QUEUED", resp.Status)
		assert.False(t, resp.SubmittedAt.IsZero())

		assert.NotNil(t, mockPub.lastMsg)
		assert.Equal(t, resp.Uuid, mockPub.lastMsg.Uuid)
		assert.Equal(t, uint(1), mockPub.lastMsg.ContentID)
		assert.Equal(t, uint(1), mockPub.lastMsg.UserID)
		assert.Equal(t, "https://git.app.uib.no/student/task1.git", mockPub.lastMsg.PayloadURL)
	})

	t.Run("401 Unauthorized when missing token", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
		rec := doReq(r, http.MethodPost, path, "", validPayload)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("400 Bad Request on invalid UUID route param", func(t *testing.T) {
		r := setupTestRouter(&mockSubmissionService{}, &mockPublisher{})
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

	t.Run("503 Service Unavailable when publisher fails", func(t *testing.T) {
		mockPub := &mockPublisher{
			publishFunc: func(ctx context.Context, msg submission.SubmissionMessage) error {
				return errors.New("rabbitmq connection lost")
			},
		}
		r := setupTestRouter(&mockSubmissionService{}, mockPub)
		rec := doReq(r, http.MethodPost, path, auth, validPayload)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}
