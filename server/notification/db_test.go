package notification

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type notificationFixture struct {
	db      *gorm.DB
	svc     INotificationService
	hub     *Hub
	teacher uuid.UUID
	alice   uuid.UUID
	bob     uuid.UUID
}

func newNotificationFixture(t *testing.T) *notificationFixture {
	t.Helper()
	db := testutil.NewSqliteTestDb(t, &user.User{}, &Notification{})
	hub := NewHub()
	f := &notificationFixture{
		db:      db,
		hub:     hub,
		svc:     NewNotificationService(db, hub, zap.NewNop().Sugar()),
		teacher: uuid.New(),
		alice:   uuid.New(),
		bob:     uuid.New(),
	}
	for i, u := range []uuid.UUID{f.teacher, f.alice, f.bob} {
		require.NoError(t, db.Create(&user.User{
			Uuid:         u,
			Username:     []string{"teacher", "alice", "bob"}[i],
			Email:        []string{"teacher@hvl.no", "alice@hvl.no", "bob@hvl.no"}[i],
			PasswordHash: "x",
			Role:         "user",
		}).Error)
	}
	return f
}

func TestSendToUsers_FansOutOnePrivateRowPerRecipient(t *testing.T) {
	f := newNotificationFixture(t)

	sent, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice, f.bob}, NotificationContent{Title: "Welcome", Body: "Course starts Monday"})
	require.NoError(t, err)
	assert.Len(t, sent, 2)

	aliceItems, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)
	require.Len(t, aliceItems, 1)
	assert.Equal(t, "Welcome", aliceItems[0].Title)
	assert.Nil(t, aliceItems[0].SeenAt)

	bobItems, err := f.svc.List(f.bob, 20, nil)
	require.NoError(t, err)
	assert.Len(t, bobItems, 1)
	assert.NotEqual(t, aliceItems[0].Uuid, bobItems[0].Uuid, "each recipient gets own row")
}

func TestSendToUsers_DeduplicatesRecipients(t *testing.T) {
	f := newNotificationFixture(t)

	sent, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice, f.alice}, NotificationContent{Title: "Hi", Body: "Once"})
	require.NoError(t, err)
	assert.Len(t, sent, 1)
}

func TestSendToUsers_UnknownRecipientWritesNothing(t *testing.T) {
	f := newNotificationFixture(t)

	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice, uuid.New()}, NotificationContent{Title: "Hi", Body: "Partial"})
	assert.ErrorIs(t, err, ErrRecipientNotFound)

	count, err := f.svc.CountUnseen(f.alice)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestList_OnlyReturnsOwnNotifications(t *testing.T) {
	f := newNotificationFixture(t)
	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.bob}, NotificationContent{Title: "For bob", Body: "Secret"})
	require.NoError(t, err)

	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestList_CursorAndCount(t *testing.T) {
	f := newNotificationFixture(t)
	for _, title := range []string{"one", "two", "three"} {
		_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: title, Body: "body"})
		require.NoError(t, err)
		time.Sleep(2 * time.Millisecond) // distinct created_at for stable ordering
	}

	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, "three", items[0].Title, "newest first")

	_, err = f.svc.MarkSeen(f.alice, items[0].Uuid)
	require.NoError(t, err)

	count, err := f.svc.CountUnseen(f.alice)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)

	before := items[1].CreatedAt
	older, err := f.svc.List(f.alice, 20, &before)
	require.NoError(t, err)
	require.Len(t, older, 1)
	assert.Equal(t, "one", older[0].Title)
}

func TestMarkSeen_IsIdempotentAndScoped(t *testing.T) {
	f := newNotificationFixture(t)
	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: "Hi", Body: "body"})
	require.NoError(t, err)
	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)

	first, err := f.svc.MarkSeen(f.alice, items[0].Uuid)
	require.NoError(t, err)
	require.NotNil(t, first.SeenAt)

	time.Sleep(5 * time.Millisecond)
	second, err := f.svc.MarkSeen(f.alice, items[0].Uuid)
	require.NoError(t, err)
	require.NotNil(t, second.SeenAt)
	assert.WithinDuration(t, *first.SeenAt, *second.SeenAt, time.Millisecond, "first seen time is kept")

	_, err = f.svc.MarkSeen(f.bob, items[0].Uuid)
	assert.ErrorIs(t, err, ErrNotificationNotFound)
}

func TestMarkAllSeen_CountsOnlyUnseen(t *testing.T) {
	f := newNotificationFixture(t)
	for i := 0; i < 3; i++ {
		_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: "t", Body: "b"})
		require.NoError(t, err)
	}

	updated, err := f.svc.MarkAllSeen(f.alice)
	require.NoError(t, err)
	assert.EqualValues(t, 3, updated)

	updated, err = f.svc.MarkAllSeen(f.alice)
	require.NoError(t, err)
	assert.Zero(t, updated)
}

func TestDelete_OnlyOwnRow(t *testing.T) {
	f := newNotificationFixture(t)
	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice, f.bob}, NotificationContent{Title: "t", Body: "b"})
	require.NoError(t, err)
	aliceItems, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)

	assert.ErrorIs(t, f.svc.Delete(f.bob, aliceItems[0].Uuid), ErrNotificationNotFound)

	require.NoError(t, f.svc.Delete(f.alice, aliceItems[0].Uuid))
	items, err := f.svc.List(f.alice, 20, nil)
	require.NoError(t, err)
	assert.Empty(t, items)

	bobItems, err := f.svc.List(f.bob, 20, nil)
	require.NoError(t, err)
	assert.Len(t, bobItems, 1, "bob's copy untouched")
}

func TestNotificationTable_HasNoUpdatedAtColumn(t *testing.T) {
	f := newNotificationFixture(t)
	assert.False(t, f.db.Migrator().HasColumn(&Notification{}, "updated_at"))
	assert.False(t, f.db.Migrator().HasColumn(&Notification{}, "deleted_at"))
}

func TestSendToUsers_PublishesToOpenStreams(t *testing.T) {
	f := newNotificationFixture(t)
	events, cancel := f.hub.Subscribe(f.alice)
	defer cancel()

	_, err := f.svc.SendToUsers(f.teacher, []uuid.UUID{f.alice}, NotificationContent{Title: "live", Body: "now"})
	require.NoError(t, err)

	select {
	case dto := <-events:
		assert.Equal(t, "live", dto.Title)
	case <-time.After(time.Second):
		t.Fatal("no event published to alice's stream")
	}
}
