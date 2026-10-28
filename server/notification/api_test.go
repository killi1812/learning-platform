package notification

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
