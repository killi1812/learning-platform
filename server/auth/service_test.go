package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func getTestDb(t *testing.T) *mongo.Database {
	uri := os.Getenv("MONGO_CONN")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("Skipping DB test: unable to connect to mongo: %v", err)
		return nil
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("Skipping DB test: mongo ping failed: %v", err)
		return nil
	}

	dbName := "lp_test_auth_" + uuid.New().String()[:8]
	db := client.Database(dbName)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
	})
	return db
}

func TestAuthService(t *testing.T) {
	app.AccessKey = "test-auth-access-key-12345"
	app.RefreshKey = "test-auth-refresh-key-12345"

	db := getTestDb(t)
	logger := zap.NewNop().Sugar()
	svc := &AuthService{db: db, logger: logger}

	// Insert test user
	userUuid := uuid.New()
	hash, err := authUtil.HashPassword("validPassword123")
	assert.NoError(t, err)

	testUser := user.User{
		Uuid:         userUuid,
		Username:     "tester",
		Email:        "tester@example.com",
		PasswordHash: hash,
		Role:         user.ROLE_USER,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = db.Collection("users").InsertOne(ctx, testUser)
	assert.NoError(t, err)

	t.Run("Login with valid username", func(t *testing.T) {
		token, refresh, err := svc.Login("tester", "validPassword123")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, refresh)

		// Verify session was inserted
		var session user.Session
		findErr := db.Collection("sessions").FindOne(ctx, bson.M{"user_uuid": userUuid}).Decode(&session)
		assert.NoError(t, findErr)
		assert.NotEmpty(t, session.RefreshToken)
	})

	t.Run("Login with valid email", func(t *testing.T) {
		token, refresh, err := svc.Login("tester@example.com", "validPassword123")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, refresh)
	})

	t.Run("Login with wrong password returns ErrInvalidCredentials", func(t *testing.T) {
		_, _, err := svc.Login("tester", "wrongPassword")
		assert.ErrorIs(t, err, cerror.ErrInvalidCredentials)
	})

	t.Run("Login with unknown identifier returns ErrInvalidCredentials", func(t *testing.T) {
		_, _, err := svc.Login("nonexistent", "validPassword123")
		assert.ErrorIs(t, err, cerror.ErrInvalidCredentials)
	})

	t.Run("Login with user having nil uuid triggers GenerateTokens error", func(t *testing.T) {
		nilUser := user.User{
			Uuid:         uuid.Nil,
			Username:     "niluser",
			Email:        "nil@example.com",
			PasswordHash: hash,
			Role:         user.ROLE_USER,
		}
		_, _ = db.Collection("users").InsertOne(ctx, nilUser)
		_, _, err := svc.Login("niluser", "validPassword123")
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with valid token", func(t *testing.T) {
		_, refresh, err := svc.Login("tester", "validPassword123")
		assert.NoError(t, err)

		newToken, newRefresh, err := svc.RefreshTokens("Bearer " + refresh)
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEmpty(t, newRefresh)
	})

	t.Run("RefreshTokens with malformed token", func(t *testing.T) {
		_, _, err := svc.RefreshTokens("invalid-token")
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with invalid uuid in token claims", func(t *testing.T) {
		// create token with bad ID
		claims := &authUtil.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "not-a-uuid"},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, _ := tok.SignedString([]byte(app.RefreshKey))
		_, _, err := svc.RefreshTokens("Bearer " + signed)
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with unknown user session", func(t *testing.T) {
		unknownUuid := uuid.New()
		_, ref, _ := authUtil.GenerateTokens("unknown@test.com", "unknown", "user", unknownUuid)
		_, _, err := svc.RefreshTokens("Bearer " + ref)
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with corrupt refresh token in session", func(t *testing.T) {
		corruptUserUuid := uuid.New()
		_, ref, _ := authUtil.GenerateTokens("c@test.com", "cuser", "user", corruptUserUuid)
		_, _ = db.Collection("sessions").InsertOne(ctx, user.Session{
			UserUuid:     corruptUserUuid,
			RefreshToken: "corrupt.refresh.token",
		})
		_, _, err := svc.RefreshTokens("Bearer " + ref)
		assert.Error(t, err)
	})

	t.Run("RefreshTokens with user not in users collection", func(t *testing.T) {
		orphanUuid := uuid.New()
		_, ref, _ := authUtil.GenerateTokens("o@test.com", "ouser", "user", orphanUuid)
		_, _ = db.Collection("sessions").InsertOne(ctx, user.Session{
			UserUuid:     orphanUuid,
			RefreshToken: ref,
		})
		_, _, err := svc.RefreshTokens("Bearer " + ref)
		assert.Error(t, err)
	})

	t.Run("Logout removes user session", func(t *testing.T) {
		err := svc.Logout(userUuid.String())
		assert.NoError(t, err)

		// Session should be gone
		var session user.Session
		findErr := db.Collection("sessions").FindOne(ctx, bson.M{"user_uuid": userUuid}).Decode(&session)
		assert.Error(t, findErr)
	})

	t.Run("Logout with invalid uuid string", func(t *testing.T) {
		err := svc.Logout("not-a-uuid")
		assert.Error(t, err)
	})
}

func TestNewAuthService(t *testing.T) {
	db := getTestDb(t)
	svc := NewAuthService(db, zap.NewNop().Sugar())
	assert.NotNil(t, svc)
}
