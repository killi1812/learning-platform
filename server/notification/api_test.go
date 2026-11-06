package notification

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	app.AccessKey = "test-notification-secret-access-key-12345"
	app.RefreshKey = "test-notification-secret-refresh-key-12345"
}

func setupNotificationRouter(t *testing.T) (*gin.Engine, *notificationFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	f := newNotificationFixture(t)
	ctn := &NotificationCtn{service: f.svc, hub: f.hub, logger: zap.NewNop().Sugar()}
	r := gin.New()
	ctn.RegisterEndpoints(r.Group("/api"))
	return r, f
}

func bearer(t *testing.T, userUuid uuid.UUID, role authutils.Role) string {
	t.Helper()
	token, _, err := authutils.GenerateTokens("user@hvl.no", "user", role, userUuid)
	require.NoError(t, err)
	return "Bearer " + token
}

func TestNotificationApi_RequiresAuth(t *testing.T) {
	r, _ := setupNotificationRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/notification", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationApi_SendIsOpenToAnyLoggedInUser(t *testing.T) {
	r, f := setupNotificationRouter(t)
	body := map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "t", "body": "b"}

	w := doRequest(r, http.MethodPost, "/api/notification/send", bearer(t, f.teacher, authutils.ROLE_USER), body)
	assert.Equal(t, http.StatusCreated, w.Code)

	w = doRequest(r, http.MethodPost, "/api/notification/send", "", body)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationApi_SendThenListAndSeen(t *testing.T) {
	r, f := setupNotificationRouter(t)
	admin := bearer(t, f.teacher, authutils.ROLE_ADMIN)
	alice := bearer(t, f.alice, authutils.ROLE_USER)

	w := doRequest(r, http.MethodPost, "/api/notification/send", admin,
		map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "Hello", "body": "World"})
	require.Equal(t, http.StatusCreated, w.Code)

	w = doRequest(r, http.MethodGet, "/api/notification", alice, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var items []NotificationDto
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Hello", items[0].Title)
	assert.Nil(t, items[0].SeenAt)

	w = doRequest(r, http.MethodGet, "/api/notification/unread-count", alice, nil)
	assert.JSONEq(t, `{"count":1}`, w.Body.String())

	w = doRequest(r, http.MethodPost, "/api/notification/"+items[0].Uuid.String()+"/seen", alice, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doRequest(r, http.MethodGet, "/api/notification/unread-count", alice, nil)
	assert.JSONEq(t, `{"count":0}`, w.Body.String())
}

func TestNotificationApi_MarkAllSeen(t *testing.T) {
	r, f := setupNotificationRouter(t)
	admin := bearer(t, f.teacher, authutils.ROLE_ADMIN)
	alice := bearer(t, f.alice, authutils.ROLE_USER)

	_ = doRequest(r, http.MethodPost, "/api/notification/send", admin,
		map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "1", "body": "1"})
	_ = doRequest(r, http.MethodPost, "/api/notification/send", admin,
		map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "2", "body": "2"})

	w := doRequest(r, http.MethodPost, "/api/notification/seen-all", alice, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"updated":2`)
}

func TestNotificationApi_ListPagingAndLimits(t *testing.T) {
	r, f := setupNotificationRouter(t)
	admin := bearer(t, f.teacher, authutils.ROLE_ADMIN)
	alice := bearer(t, f.alice, authutils.ROLE_USER)

	_ = doRequest(r, http.MethodPost, "/api/notification/send", admin,
		map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "1", "body": "1"})
	time.Sleep(2 * time.Millisecond)
	_ = doRequest(r, http.MethodPost, "/api/notification/send", admin,
		map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "2", "body": "2"})

	w := doRequest(r, http.MethodGet, "/api/notification?limit=1", alice, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var items []NotificationDto
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)
	assert.Equal(t, "2", items[0].Title)

	before := url.QueryEscape(items[0].CreatedAt.Format(time.RFC3339Nano))
	w = doRequest(r, http.MethodGet, "/api/notification?before="+before, alice, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var older []NotificationDto
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &older))
	require.Len(t, older, 1)
	assert.Equal(t, "1", older[0].Title)

	// Invalid limit and before
	w = doRequest(r, http.MethodGet, "/api/notification?limit=abc", alice, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = doRequest(r, http.MethodGet, "/api/notification?before=invalid", alice, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid UUIDs for seen and delete
	w = doRequest(r, http.MethodPost, "/api/notification/not-a-uuid/seen", alice, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = doRequest(r, http.MethodDelete, "/api/notification/not-a-uuid", alice, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Send invalid payload
	w = doRequest(r, http.MethodPost, "/api/notification/send", admin, map[string]any{"title": ""})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationCtn_Constructor(t *testing.T) {
	f := newNotificationFixture(t)
	ctn := NewNotificationCtn(f.svc, f.hub, zap.NewNop().Sugar())
	assert.NotNil(t, ctn)
}

func TestNotificationApi_HandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctn := &NotificationCtn{logger: zap.NewNop().Sugar()}

	testCases := []struct {
		err      error
		expected int
	}{
		{err: nil, expected: 200},
		{err: ErrNotificationNotFound, expected: 404},
		{err: ErrRecipientNotFound, expected: 404},
		{err: ErrUserNotFound, expected: 401},
		{err: assert.AnError, expected: 500},
	}

	for _, tc := range testCases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		handled := ctn.handleError(c, tc.err, "fallback")
		if tc.err == nil {
			assert.False(t, handled)
		} else {
			assert.True(t, handled)
			assert.Equal(t, tc.expected, w.Code)
		}
	}
}

type mockNotificationServiceForApi struct {
	INotificationService
	countUnseenErr error
	markSeenErr    error
	markAllSeenErr error
	deleteErr      error
}

func (m *mockNotificationServiceForApi) CountUnseen(userUuid uuid.UUID) (int64, error) {
	if m.countUnseenErr != nil {
		return 0, m.countUnseenErr
	}
	return m.INotificationService.CountUnseen(userUuid)
}

func (m *mockNotificationServiceForApi) MarkSeen(userUuid, notificationUuid uuid.UUID) (*Notification, error) {
	if m.markSeenErr != nil {
		return nil, m.markSeenErr
	}
	return m.INotificationService.MarkSeen(userUuid, notificationUuid)
}

func (m *mockNotificationServiceForApi) MarkAllSeen(userUuid uuid.UUID) (int64, error) {
	if m.markAllSeenErr != nil {
		return 0, m.markAllSeenErr
	}
	return m.INotificationService.MarkAllSeen(userUuid)
}

func (m *mockNotificationServiceForApi) Delete(userUuid, notificationUuid uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.INotificationService.Delete(userUuid, notificationUuid)
}

func TestNotificationApi_ServiceErrorResponses(t *testing.T) {
	f := newNotificationFixture(t)
	mock := &mockNotificationServiceForApi{
		INotificationService: f.svc,
		countUnseenErr:       assert.AnError,
		markSeenErr:          ErrNotificationNotFound,
		markAllSeenErr:       assert.AnError,
		deleteErr:            ErrNotificationNotFound,
	}
	ctn := &NotificationCtn{service: mock, hub: f.hub, logger: zap.NewNop().Sugar()}
	r := gin.New()
	ctn.RegisterEndpoints(r.Group("/api"))

	userToken := bearer(t, f.alice, authutils.ROLE_USER)

	w := doRequest(r, http.MethodGet, "/api/notification/unread-count", userToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = doRequest(r, http.MethodPost, "/api/notification/"+uuid.New().String()+"/seen", userToken, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = doRequest(r, http.MethodPost, "/api/notification/seen-all", userToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = doRequest(r, http.MethodDelete, "/api/notification/"+uuid.New().String(), userToken, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotificationApi_DeleteForeignIsNotFound(t *testing.T) {
	r, f := setupNotificationRouter(t)
	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: "t", Body: "b"})
	require.NoError(t, err)
	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)

	w := doRequest(r, http.MethodDelete, "/api/notification/"+items[0].Uuid.String(), bearer(t, f.bob, authutils.ROLE_USER), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = doRequest(r, http.MethodDelete, "/api/notification/"+items[0].Uuid.String(), bearer(t, f.alice, authutils.ROLE_USER), nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestNotificationApi_StreamDeliversPublishedEvent(t *testing.T) {
	// Real server: c.Stream needs http.CloseNotifier, which httptest.ResponseRecorder lacks.
	r, f := setupNotificationRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/notification/stream", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", bearer(t, f.alice, authutils.ROLE_USER))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	// Wait for the stream to subscribe before publishing.
	require.Eventually(t, func() bool {
		f.hub.mu.Lock()
		defer f.hub.mu.Unlock()
		return len(f.hub.subs[f.alice]) == 1
	}, time.Second, 5*time.Millisecond)

	_, err = f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: "live", Body: "event"})
	require.NoError(t, err)

	lines := make(chan string)
	readerDone := make(chan struct{})
	defer close(readerDone)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-readerDone:
				return
			}
		}
	}()

	var got strings.Builder
	timeout := time.After(2 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("stream closed before event, got:\n%s", got.String())
			}
			got.WriteString(line + "\n")
			if strings.Contains(got.String(), `"title":"live"`) {
				// Gin writes "event:notification" without a space; SSE accepts both forms.
				assert.Contains(t, got.String(), "event:notification")
				return
			}
		case <-timeout:
			t.Fatalf("no notification event received, got:\n%s", got.String())
		}
	}
}

func doRequest(r http.Handler, method, path, auth string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
