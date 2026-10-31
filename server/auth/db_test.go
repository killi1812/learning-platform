package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func getTestMongoDb(t *testing.T) *mongo.Database {
	candidates := []string{}
	if uri := os.Getenv("MONGO_CONN"); uri != "" {
		candidates = append(candidates, uri)
	}
	candidates = append(candidates, "mongodb://localhost:27018", "mongodb://localhost:27017")

	var client *mongo.Client
	var err error
	for _, uri := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		c, connErr := mongo.Connect(options.Client().ApplyURI(uri))
		if connErr == nil && c.Ping(ctx, nil) == nil {
			cancel()
			client = c
			break
		}
		cancel()
		if c != nil {
			_ = c.Disconnect(context.Background())
		}
		err = connErr
	}

	if client == nil {
		t.Skipf("Skipping Mongo test: unable to connect to mongo (tried %v): %v", candidates, err)
		return nil
	}

	dbName := "lp_test_auth_" + uuid.New().String()[:8]
	db := client.Database(dbName)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
	})
	return db
}

func TestAuthService_Unit(t *testing.T) {
	app.AccessKey = "test-auth-access-key-12345"
	app.RefreshKey = "test-auth-refresh-key-12345"

	gormDb := testutil.NewSqliteTestDb(t, &user.User{}, &notification.Notification{})
	logger := zap.NewNop().Sugar()
	svc := &AuthService{
		db:            gormDb,
		logger:        logger,
		notifications: notification.NewNotificationService(gormDb, notification.NewHub(), logger),
	}

	// Insert test user
	userUuid := uuid.New()
	hash, err := authUtil.HashPassword("validPassword123")
	assert.NoError(t, err)

	testUser := user.User{
		Uuid:         userUuid,
		Username:     "tester",
		Email:        "tester@example.com",
		PasswordHash: hash,
		Role:         authUtil.ROLE_USER,
	}

	err = gormDb.Create(&testUser).Error
	assert.NoError(t, err)

	t.Run("Login with unknown email returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := svc.Login("nonexistent@example.com", "validPassword123", "", "", "")
		assert.ErrorIs(t, err, cerror.ErrInvalidCredentials)
	})

	t.Run("Login with wrong password returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := svc.Login("tester@example.com", "wrongPassword", "", "", "")
		assert.ErrorIs(t, err, cerror.ErrInvalidCredentials)
	})

	t.Run("Login is case-insensitive for email", func(t *testing.T) {
		tokens1, err := svc.Login("TESTER@EXAMPLE.COM", "validPassword123", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens1.AccessToken)
		assert.NotEmpty(t, tokens1.RefreshToken)
		assert.NotEmpty(t, tokens1.SessionId)

		tokens2, err := svc.Login("Tester@Example.com", "validPassword123", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens2.AccessToken)
		assert.NotEmpty(t, tokens2.RefreshToken)
		assert.NotEmpty(t, tokens2.SessionId)
	})

	t.Run("RefreshTokens with malformed token", func(t *testing.T) {
		_, err := svc.RefreshTokens("invalid-token")
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with invalid uuid in token claims", func(t *testing.T) {
		claims := &authUtil.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "not-a-uuid"},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, _ := tok.SignedString([]byte(app.RefreshKey))
		_, err := svc.RefreshTokens("Bearer " + signed)
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with nil mongoDb returns error", func(t *testing.T) {
		_, validRefresh, err := authUtil.GenerateTokens("tester@example.com", "tester", authUtil.ROLE_USER, userUuid)
		assert.NoError(t, err)
		_, err = svc.RefreshTokens(validRefresh)
		assert.Error(t, err)
	})

	t.Run("LogoutCurrent with nil mongoDb returns nil", func(t *testing.T) {
		err := svc.LogoutCurrent(uuid.New(), uuid.New())
		assert.NoError(t, err)
	})

	t.Run("LogoutAll with nil mongoDb returns nil", func(t *testing.T) {
		err := svc.LogoutAll(uuid.New())
		assert.NoError(t, err)
	})

	t.Run("ListSessions with nil mongoDb returns empty slice", func(t *testing.T) {
		sessions, err := svc.ListSessions(uuid.New(), uuid.New())
		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("RevokeSession with nil mongoDb returns nil", func(t *testing.T) {
		err := svc.RevokeSession(uuid.New(), uuid.New())
		assert.NoError(t, err)
	})

	t.Run("RegisterSessionValidator with nil mongoDb does nothing", func(t *testing.T) {
		svc.RegisterSessionValidator()
	})

	t.Run("Login with invalid requestedSessionId falls back to new session", func(t *testing.T) {
		tokens, err := svc.Login("tester@example.com", "validPassword123", "Test Browser", "127.0.0.1", "invalid-uuid-string")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.SessionId)
	})

	t.Run("Register new user succeeds and normalizes lowercase", func(t *testing.T) {
		tokens, err := svc.Register("BrandNew", "BrandNew@Example.Com", "Password123", "Mozilla/5.0", "127.0.0.1")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.NotEmpty(t, tokens.SessionId)

		var created user.User
		err = gormDb.Where("username = ?", "brandnew").First(&created).Error
		assert.NoError(t, err)
		assert.Equal(t, "brandnew@example.com", created.Email)
		assert.True(t, authUtil.VerifyPassword(created.PasswordHash, "Password123"))
	})

	t.Run("Register duplicate username returns error", func(t *testing.T) {
		_, err := svc.Register("tester", "other@example.com", "Password123", "", "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cerror.ErrUserAlreadyExists)
	})

	t.Run("Register duplicate username with differing case returns error", func(t *testing.T) {
		_, err := svc.Register("TESTER", "other2@example.com", "Password123", "", "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cerror.ErrUserAlreadyExists)
	})

	t.Run("Register duplicate email returns error", func(t *testing.T) {
		_, err := svc.Register("uniqueusername", "tester@example.com", "Password123", "", "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cerror.ErrUserAlreadyExists)
	})

	t.Run("Login returns error when database query fails", func(t *testing.T) {
		closedDb := testutil.NewSqliteTestDb(t, &user.User{})
		sqlDb, _ := closedDb.DB()
		_ = sqlDb.Close()
		failingSvc := &AuthService{db: closedDb, logger: logger}
		_, err := failingSvc.Login("tester@example.com", "pass", "", "", "")
		assert.Error(t, err)
	})

	t.Run("Register returns error when database query fails", func(t *testing.T) {
		closedDb := testutil.NewSqliteTestDb(t, &user.User{})
		sqlDb, _ := closedDb.DB()
		_ = sqlDb.Close()
		failingSvc := &AuthService{db: closedDb, logger: logger}
		_, err := failingSvc.Register("newuser", "new@example.com", "pass", "", "")
		assert.Error(t, err)
	})

	t.Run("Register succeeds even when notification delivery fails", func(t *testing.T) {
		svcWithFailingNotifier := &AuthService{
			db:            gormDb,
			logger:        logger,
			notifications: &mockFailingNotifier{},
		}
		tokens, err := svcWithFailingNotifier.Register("notifieruser", "notifier@example.com", "Password123", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
	})
}

type mockFailingNotifier struct {
	notification.INotificationService
}

func (m *mockFailingNotifier) NotifySystem(recipientUuid uuid.UUID, content notification.NotificationContent) (*notification.Notification, error) {
	return nil, errors.New("notification delivery error")
}

func TestAuthService_Integration(t *testing.T) {
	origAccess := app.AccessKey
	origRefresh := app.RefreshKey
	t.Cleanup(func() {
		app.AccessKey = origAccess
		app.RefreshKey = origRefresh
	})
	app.AccessKey = "test-auth-access-key-12345"
	app.RefreshKey = "test-auth-refresh-key-12345"

	mongoDb := getTestMongoDb(t)
	gormDb := testutil.NewSqliteTestDb(t, &user.User{}, &notification.Notification{})
	logger := zap.NewNop().Sugar()
	svc := &AuthService{
		db:            gormDb,
		mongoDb:       mongoDb,
		logger:        logger,
		notifications: notification.NewNotificationService(gormDb, notification.NewHub(), logger),
	}

	// Insert test user
	userUuid := uuid.New()
	hash, err := authUtil.HashPassword("validPassword123")
	assert.NoError(t, err)

	testUser := user.User{
		Uuid:         userUuid,
		Username:     "tester",
		Email:        "tester@example.com",
		PasswordHash: hash,
		Role:         authUtil.ROLE_USER,
	}

	err = gormDb.Create(&testUser).Error
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var firstSessionUuid uuid.UUID
	var firstRefreshToken string

	t.Run("Login with valid email stores hashed token and device name", func(t *testing.T) {
		ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36"
		tokens, err := svc.Login("tester@example.com", "validPassword123", ua, "192.168.1.10", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.NotEmpty(t, tokens.SessionId)

		firstRefreshToken = tokens.RefreshToken

		// Verify session was inserted in Mongo
		var claims authUtil.Claims
		_, _ = jwt.ParseWithClaims(tokens.AccessToken, &claims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})
		firstSessionUuid = claims.SessionUuid
		assert.Equal(t, firstSessionUuid.String(), tokens.SessionId)

		var session user.Session
		findErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{"session_uuid": firstSessionUuid}).Decode(&session)
		assert.NoError(t, findErr)
		assert.Equal(t, hashToken(tokens.RefreshToken), session.RefreshToken)
		assert.Equal(t, "Chrome on Linux", session.DeviceName)
		assert.Equal(t, "192.168.1.10", session.IpAddress)
	})

	t.Run("ListSessions returns active sessions with current indicator", func(t *testing.T) {
		sessions, err := svc.ListSessions(userUuid, firstSessionUuid)
		assert.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, firstSessionUuid.String(), sessions[0].Id)
		assert.Equal(t, "Chrome on Linux", sessions[0].DeviceName)
		assert.True(t, sessions[0].IsCurrent)
	})

	t.Run("RefreshTokens preserves SessionUuid and rotates TokenUuid and hashed refresh token", func(t *testing.T) {
		newTokens, err := svc.RefreshTokens("Bearer " + firstRefreshToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)

		var newClaims authUtil.Claims
		_, _ = jwt.ParseWithClaims(newTokens.AccessToken, &newClaims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})
		assert.Equal(t, firstSessionUuid, newClaims.SessionUuid)

		// Verify mongo was updated with new token hash
		var session user.Session
		findErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{"session_uuid": firstSessionUuid}).Decode(&session)
		assert.NoError(t, findErr)
		assert.Equal(t, hashToken(newTokens.RefreshToken), session.RefreshToken)

		// Token reuse detection test: using the old firstRefreshToken should be rejected and revoke session
		_, reuseErr := svc.RefreshTokens("Bearer " + firstRefreshToken)
		assert.Error(t, reuseErr)

		// Session should have been deleted due to reuse detection
		var revokedSession user.Session
		revokedErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{"session_uuid": firstSessionUuid}).Decode(&revokedSession)
		assert.Error(t, revokedErr)
	})

	t.Run("Multiple logins create separate sessions without destroying previous", func(t *testing.T) {
		_, err1 := svc.Login("tester@example.com", "validPassword123", "PostmanRuntime/7.36.0", "10.0.0.1", "")
		assert.NoError(t, err1)
		_, err2 := svc.Login("tester@example.com", "validPassword123", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1", "10.0.0.2", "")
		assert.NoError(t, err2)

		sessions, err := svc.ListSessions(userUuid, uuid.New())
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(sessions), 2)

		// Revoke one session
		targetId, _ := uuid.Parse(sessions[0].Id)
		err = svc.RevokeSession(userUuid, targetId)
		assert.NoError(t, err)

		remainingSessions, err := svc.ListSessions(userUuid, uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, len(sessions)-1, len(remainingSessions))

		// LogoutAll removes all sessions
		err = svc.LogoutAll(userUuid)
		assert.NoError(t, err)

		finalSessions, err := svc.ListSessions(userUuid, uuid.New())
		assert.NoError(t, err)
		assert.Empty(t, finalSessions)
	})

	t.Run("Re-login with expired session UUID reactivates existing session document without duplicate", func(t *testing.T) {
		// Clean up any remaining sessions
		_ = svc.LogoutAll(userUuid)

		// First login
		loginTokens, err := svc.Login("tester@example.com", "validPassword123", "Chrome on Linux", "127.0.0.1", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, loginTokens.AccessToken)
		parsedSessUuid, err := uuid.Parse(loginTokens.SessionId)
		assert.NoError(t, err)

		// Verify 1 active session in DB
		count, err := mongoDb.Collection("sessions").CountDocuments(ctx, bson.M{"user_uuid": userUuid})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Simulate session expiration in Mongo (expires_at in the past)
		pastTime := time.Now().Add(-2 * time.Hour)
		_, err = mongoDb.Collection("sessions").UpdateOne(ctx,
			bson.M{"session_uuid": parsedSessUuid},
			bson.M{"$set": bson.M{"expires_at": pastTime}},
		)
		assert.NoError(t, err)

		// ListSessions should filter out expired session
		activeSessions, err := svc.ListSessions(userUuid, uuid.New())
		assert.NoError(t, err)
		assert.Empty(t, activeSessions, "Expired session should not be returned by ListSessions")

		// Re-login passing the persistent sessionId
		loginTokens2, err := svc.Login("tester@example.com", "validPassword123", "Chrome on Linux", "127.0.0.1", loginTokens.SessionId)
		assert.NoError(t, err)
		assert.NotEmpty(t, loginTokens2.AccessToken)
		assert.Equal(t, loginTokens.SessionId, loginTokens2.SessionId, "Should reuse the existing session UUID")

		// Verify that NO DUPLICATE document was created in Mongo
		countAfter, err := mongoDb.Collection("sessions").CountDocuments(ctx, bson.M{"user_uuid": userUuid})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), countAfter, "Should still have exactly 1 session document in Mongo (no duplicate)")

		// Verify ListSessions now shows the reactivated session
		reactivatedSessions, err := svc.ListSessions(userUuid, parsedSessUuid)
		assert.NoError(t, err)
		assert.Len(t, reactivatedSessions, 1)
		assert.Equal(t, loginTokens.SessionId, reactivatedSessions[0].Id)
		assert.True(t, reactivatedSessions[0].IsCurrent)
		assert.True(t, reactivatedSessions[0].ExpiresAt.After(time.Now()))
	})

	t.Run("SessionValidator rejects token immediately when session is deleted or revoked", func(t *testing.T) {
		svc.RegisterSessionValidator()
		defer authUtil.SetSessionValidator(nil)

		loginTokens3, err := svc.Login("tester@example.com", "validPassword123", "Test Browser", "127.0.0.1", "")
		assert.NoError(t, err)
		parsedSessUuid, err := uuid.Parse(loginTokens3.SessionId)
		assert.NoError(t, err)

		// Parse token claims
		var claims authUtil.Claims
		_, _ = jwt.ParseWithClaims(loginTokens3.AccessToken, &claims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})

		// Test the session validator through Protect middleware
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/test-protected", authUtil.Protect(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		reqActive := httptest.NewRequest(http.MethodGet, "/test-protected", nil)
		reqActive.Header.Set("Authorization", "Bearer "+loginTokens3.AccessToken)
		wActive := httptest.NewRecorder()
		r.ServeHTTP(wActive, reqActive)
		assert.Equal(t, http.StatusOK, wActive.Code)

		var session user.Session
		findErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{
			"user_uuid":    userUuid,
			"session_uuid": parsedSessUuid,
			"expires_at":   bson.M{"$gt": time.Now()},
		}).Decode(&session)
		assert.NoError(t, findErr)

		// Revoke session
		err = svc.RevokeSession(userUuid, parsedSessUuid)
		assert.NoError(t, err)

		// After revoke, Protect middleware rejects with 401
		reqRevoked := httptest.NewRequest(http.MethodGet, "/test-protected", nil)
		reqRevoked.Header.Set("Authorization", "Bearer "+loginTokens3.AccessToken)
		wRevoked := httptest.NewRecorder()
		r.ServeHTTP(wRevoked, reqRevoked)
		assert.Equal(t, http.StatusUnauthorized, wRevoked.Code)
		findErrAfter := mongoDb.Collection("sessions").FindOne(ctx, bson.M{
			"user_uuid":    userUuid,
			"session_uuid": parsedSessUuid,
			"expires_at":   bson.M{"$gt": time.Now()},
		}).Decode(&session)
		assert.ErrorIs(t, findErrAfter, mongo.ErrNoDocuments)
	})

	t.Run("Register with mongoDb inserts user and session document", func(t *testing.T) {
		res, err := svc.Register("mongoreg", "mongoreg@example.com", "Password123", "Firefox on Linux", "10.0.0.5")
		assert.NoError(t, err)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.RefreshToken)
		assert.NotEmpty(t, res.SessionId)

		sessUuid, err := uuid.Parse(res.SessionId)
		assert.NoError(t, err)

		var session user.Session
		findErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{"session_uuid": sessUuid}).Decode(&session)
		assert.NoError(t, findErr)
		assert.Equal(t, "Web Browser on Linux", session.DeviceName)
		assert.Equal(t, "10.0.0.5", session.IpAddress)
	})

	t.Run("Login with requestedSessionId belonging to different user generates new session", func(t *testing.T) {
		otherUserUuid := uuid.New()
		otherSessionUuid := uuid.New()
		_, err := mongoDb.Collection("sessions").InsertOne(ctx, user.Session{
			SessionUuid:  otherSessionUuid,
			UserUuid:     otherUserUuid,
			RefreshToken: "some-hash",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		})
		assert.NoError(t, err)

		tokens, err := svc.Login("tester@example.com", "validPassword123", "Test Browser", "127.0.0.1", otherSessionUuid.String())
		assert.NoError(t, err)
		assert.NotEqual(t, otherSessionUuid.String(), tokens.SessionId, "Should generate a new session UUID instead of hijacking another user's session")
	})

	t.Run("LogoutCurrent deletes only specific session from Mongo", func(t *testing.T) {
		loginTokens, err := svc.Login("tester@example.com", "validPassword123", "Chrome", "127.0.0.1", "")
		assert.NoError(t, err)
		sessUuid, err := uuid.Parse(loginTokens.SessionId)
		assert.NoError(t, err)

		err = svc.LogoutCurrent(userUuid, sessUuid)
		assert.NoError(t, err)

		var session user.Session
		findErr := mongoDb.Collection("sessions").FindOne(ctx, bson.M{"session_uuid": sessUuid}).Decode(&session)
		assert.ErrorIs(t, findErr, mongo.ErrNoDocuments)
	})

	t.Run("NewAuthService with MongoDb registers session validator", func(t *testing.T) {
		fullSvc := NewAuthService(AuthServiceParams{
			GormDb:  gormDb,
			MongoDb: mongoDb,
			Logger:  logger,
		})
		assert.NotNil(t, fullSvc)
	})

	t.Run("RefreshTokens fails when session document not found in mongoDb", func(t *testing.T) {
		orphanSessionUuid := uuid.New()
		_, refreshToken, err := authUtil.GenerateTokens(testUser.Email, testUser.Username, testUser.Role, testUser.Uuid, orphanSessionUuid)
		assert.NoError(t, err)

		_, err = svc.RefreshTokens(refreshToken)
		assert.ErrorIs(t, err, cerror.ErrInvalidTokenFormat)
	})

	t.Run("RefreshTokens fails when user not found in gorm db", func(t *testing.T) {
		missingUserUuid := uuid.New()
		missingSessionUuid := uuid.New()
		accessToken, refreshToken, err := authUtil.GenerateTokens("missing@example.com", "missing", authUtil.ROLE_USER, missingUserUuid, missingSessionUuid)
		assert.NoError(t, err)

		var claims authUtil.Claims
		_, _ = jwt.ParseWithClaims(accessToken, &claims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})

		_, err = mongoDb.Collection("sessions").InsertOne(ctx, user.Session{
			SessionUuid:  missingSessionUuid,
			UserUuid:     missingUserUuid,
			TokenUuid:    claims.TokenUuid,
			RefreshToken: hashToken(refreshToken),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		})
		assert.NoError(t, err)

		_, err = svc.RefreshTokens(refreshToken)
		assert.Error(t, err)
	})

	t.Run("LogoutCurrent, LogoutAll, ListSessions, RevokeSession return error when mongoDb disconnected", func(t *testing.T) {
		disconnectedClient, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27018"))
		assert.NoError(t, err)
		disconnectedDb := disconnectedClient.Database("lp_disconnected_" + uuid.New().String()[:8])
		_ = disconnectedClient.Disconnect(context.Background())

		failingMongoSvc := &AuthService{
			db:      gormDb,
			mongoDb: disconnectedDb,
			logger:  logger,
		}

		assert.Error(t, failingMongoSvc.LogoutCurrent(userUuid, uuid.New()))
		assert.Error(t, failingMongoSvc.LogoutAll(userUuid))
		_, err = failingMongoSvc.ListSessions(userUuid, uuid.New())
		assert.Error(t, err)
		assert.Error(t, failingMongoSvc.RevokeSession(userUuid, uuid.New()))
	})
}

func TestNewAuthService(t *testing.T) {
	gormDb := testutil.NewSqliteTestDb(t, &user.User{})
	svc := NewAuthService(AuthServiceParams{
		GormDb:  gormDb,
		MongoDb: nil,
		Logger:  zap.NewNop().Sugar(),
	})
	assert.NotNil(t, svc)
}

func TestAuthService_RefreshTokens_NilMongoDb(t *testing.T) {
	gormDb := testutil.NewSqliteTestDb(t, &user.User{})
	svc := NewAuthService(AuthServiceParams{
		GormDb:  gormDb,
		MongoDb: nil,
		Logger:  zap.NewNop().Sugar(),
	})

	userUuid := uuid.New()
	sessionUuid := uuid.New()
	_, validRefreshToken, err := authUtil.GenerateTokens("test@example.com", "testuser", authUtil.ROLE_USER, userUuid, sessionUuid)
	assert.NoError(t, err)

	tokens, err := svc.RefreshTokens("Bearer " + validRefreshToken)
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Equal(t, "database connection unavailable", err.Error())
}

func TestAuthService_RefreshTokens_AccessTokenRejected(t *testing.T) {
	app.RefreshKey = "test-refresh-key"
	gormDb := testutil.NewSqliteTestDb(t, &user.User{})
	svc := NewAuthService(AuthServiceParams{
		GormDb:  gormDb,
		MongoDb: nil,
		Logger:  zap.NewNop().Sugar(),
	})

	// Create a token with TokenTypeAccess signed with RefreshKey
	accessClaims := authUtil.Claims{
		TokenType: authUtil.TokenTypeAccess,
		Username:  "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenStr, _ := token.SignedString([]byte(app.RefreshKey))

	tokens, err := svc.RefreshTokens("Bearer " + tokenStr)
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, cerror.ErrInvalidTokenFormat)
}
