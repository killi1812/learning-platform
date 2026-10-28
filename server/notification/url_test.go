package notification

import (
	"net/http"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendToUsers_StoresURL(t *testing.T) {
	f := newNotificationFixture(t)
	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, "Course", "New module", "/courses/abc")
	require.NoError(t, err)

	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "/courses/abc", items[0].URL)
}

func TestNotificationApi_SendRejectsExternalURL(t *testing.T) {
	r, f := setupNotificationRouter(t)
	admin := bearer(t, f.teacher, authutils.ROLE_ADMIN)

	for _, url := range []string{"https://evil.example", "//evil.example", "courses/abc"} {
		body := map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "t", "body": "b", "url": url}
		w := doRequest(r, http.MethodPost, "/api/notification/send", admin, body)
		assert.Equal(t, http.StatusBadRequest, w.Code, "url %q should be rejected", url)
	}
}

func TestNotificationApi_SendAcceptsInternalURL(t *testing.T) {
	r, f := setupNotificationRouter(t)
	admin := bearer(t, f.teacher, authutils.ROLE_ADMIN)

	body := map[string]any{"userUuids": []uuid.UUID{f.alice}, "title": "t", "body": "b", "url": "/courses/abc"}
	w := doRequest(r, http.MethodPost, "/api/notification/send", admin, body)
	assert.Equal(t, http.StatusCreated, w.Code)
}
