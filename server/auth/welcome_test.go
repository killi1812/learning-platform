package auth

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRegister_SendsWelcomeNotification(t *testing.T) {
	app.AccessKey = "test-auth-access-key-12345"
	app.RefreshKey = "test-auth-refresh-key-12345"

	gormDb := testutil.NewSqliteTestDb(t, &user.User{}, &notification.Notification{})
	logger := zap.NewNop().Sugar()
	svc := &AuthService{
		db:            gormDb,
		logger:        logger,
		notifications: notification.NewNotificationService(gormDb, notification.NewHub(), logger),
	}

	_, err := svc.Register("welcome", "welcome@example.com", "Password123", "", "")
	require.NoError(t, err)

	var usr user.User
	require.NoError(t, gormDb.Where("username = ?", "welcome").First(&usr).Error)

	var rows []notification.Notification
	require.NoError(t, gormDb.Where("user_id = ?", usr.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, welcomeContent.Title, rows[0].Title)
	assert.Equal(t, welcomeContent.Body, rows[0].Body)
	assert.Equal(t, welcomeContent.URL, rows[0].URL)
	assert.Nil(t, rows[0].SenderID, "welcome message has no sender")
	assert.Nil(t, rows[0].SeenAt)
}
