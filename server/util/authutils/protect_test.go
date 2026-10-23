package authutils_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MiddlewareTestSuite struct {
	suite.Suite
	router *gin.Engine
}

func (suite *MiddlewareTestSuite) SetupSuite() {
	app.AccessKey = "test-middleware-access-key"
	app.RefreshKey = "test-middleware-refresh-key"

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	suite.router.GET("/protected/general", authutils.Protect(), func(c *gin.Context) {
		c.String(http.StatusOK, "general_access_granted")
	})

	suite.router.GET("/protected/admin", authutils.Protect("admin"), func(c *gin.Context) {
		c.String(http.StatusOK, "admin_access_granted")
	})
}

func (suite *MiddlewareTestSuite) performRequest(method, path, token string, body ...string) *httptest.ResponseRecorder {
	var reqBody *strings.Reader
	if len(body) > 0 {
		reqBody = strings.NewReader(body[0])
	} else {
		reqBody = strings.NewReader("")
	}

	req, _ := http.NewRequest(method, path, reqBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *MiddlewareTestSuite) generateToken(userID, username string, userRole string, expiresAt time.Time) string {
	claims := &authutils.Claims{
		Username: username,
		Role:     authutils.Role(userRole),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(app.AccessKey))
	if err != nil {
		suite.T().Fatalf("Failed to sign token for test: %v", err)
	}
	return tokenString
}

func (suite *MiddlewareTestSuite) TestProtect_ValidToken_GeneralAccess() {
	testUserID := "123"
	token := suite.generateToken(testUserID, "test@example.com", "", time.Now().Add(5*time.Minute))

	w := suite.performRequest(http.MethodGet, "/protected/general", token)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "general_access_granted", w.Body.String())
}

func (suite *MiddlewareTestSuite) TestProtect_NoToken() {
	w := suite.performRequest(http.MethodGet, "/protected/general", "")

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Missing token")
}

func (suite *MiddlewareTestSuite) TestProtect_InvalidTokenFormat_NoBearer() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected/general", nil)
	req.Header.Set("Authorization", "InvalidTokenWithoutBearerPrefix")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid token format")
}

func (suite *MiddlewareTestSuite) TestProtect_InvalidTokenFormat_TooShort() {
	w := suite.performRequest(http.MethodGet, "/protected/general", "short")

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid token format")
}

func (suite *MiddlewareTestSuite) TestProtect_MalformedToken() {
	malformedToken := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected/general", nil)
	req.Header.Set("Authorization", malformedToken)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid token")
}

func (suite *MiddlewareTestSuite) TestProtect_ExpiredToken() {
	testUserID := "1234"
	expiredToken := suite.generateToken(testUserID, "expired@example.com", "", time.Now().Add(-5*time.Minute))

	w := suite.performRequest(http.MethodGet, "/protected/general", expiredToken)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid token")
}

func (suite *MiddlewareTestSuite) TestProtect_WrongSigningKey() {
	claims := &authutils.Claims{
		Username: "wrongkey@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "12345",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	wrongKeyToken, _ := token.SignedString([]byte("a-completely-different-secret-key"))

	w := suite.performRequest(http.MethodGet, "/protected/general", wrongKeyToken)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid token")
}

func (suite *MiddlewareTestSuite) TestProtect_ValidToken_CorrectRole() {
	testUserUUID := "123456"
	adminToken := suite.generateToken(testUserUUID, "admin@example.com", "admin", time.Now().Add(5*time.Minute))

	w := suite.performRequest(http.MethodGet, "/protected/admin", adminToken)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "admin_access_granted", w.Body.String())
}

func (suite *MiddlewareTestSuite) TestProtect_ValidToken_InsufficientRole() {
	testUserUUID := "1234567"
	userToken := suite.generateToken(testUserUUID, "user@example.com", "", time.Now().Add(5*time.Minute))

	w := suite.performRequest(http.MethodGet, "/protected/admin", userToken)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *MiddlewareTestSuite) TestProtect_SessionValidator() {
	testUserUUID := uuid.New()
	activeSessionUUID := uuid.New()
	revokedSessionUUID := uuid.New()

	tokenActive, _, err := authutils.GenerateTokens("active@test.com", "activeuser", authutils.ROLE_USER, testUserUUID, activeSessionUUID)
	assert.NoError(suite.T(), err)

	tokenRevoked, _, err := authutils.GenerateTokens("revoked@test.com", "revokeduser", authutils.ROLE_USER, testUserUUID, revokedSessionUUID)
	assert.NoError(suite.T(), err)

	authutils.SetSessionValidator(func(ctx context.Context, uUuid, sUuid uuid.UUID) bool {
		return sUuid == activeSessionUUID
	})
	defer authutils.SetSessionValidator(nil)

	// Active session succeeds
	wActive := suite.performRequest(http.MethodGet, "/protected/general", tokenActive)
	assert.Equal(suite.T(), http.StatusOK, wActive.Code)
	assert.Equal(suite.T(), "general_access_granted", wActive.Body.String())

	// Revoked session rejected with 401
	wRevoked := suite.performRequest(http.MethodGet, "/protected/general", tokenRevoked)
	assert.Equal(suite.T(), http.StatusUnauthorized, wRevoked.Code)
	assert.Contains(suite.T(), wRevoked.Body.String(), "Session revoked or expired")
}

func (suite *MiddlewareTestSuite) TestCheckCourseAccess() {
	notFoundErr := errors.New("course not found")
	validUserUuid := uuid.New()
	courseUuid := uuid.New()

	// 1. Missing claims -> 401
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	ok1 := authutils.CheckCourseAccess(c1, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return true, nil
	}, notFoundErr)
	assert.False(suite.T(), ok1)
	assert.Equal(suite.T(), http.StatusUnauthorized, w1.Code)

	// 2. Invalid user UUID in claims -> 401
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	badClaims := &authutils.Claims{Role: authutils.ROLE_USER}
	badClaims.ID = "not-a-uuid"
	c2.Set(authutils.ContextClaimsKey, badClaims)
	ok2 := authutils.CheckCourseAccess(c2, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return true, nil
	}, notFoundErr)
	assert.False(suite.T(), ok2)
	assert.Equal(suite.T(), http.StatusUnauthorized, w2.Code)

	validClaims := &authutils.Claims{Role: authutils.ROLE_USER}
	validClaims.ID = validUserUuid.String()

	// 3. NotFound error -> 404
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Set(authutils.ContextClaimsKey, validClaims)
	ok3 := authutils.CheckCourseAccess(c3, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return false, notFoundErr
	}, notFoundErr)
	assert.False(suite.T(), ok3)
	assert.Equal(suite.T(), http.StatusNotFound, w3.Code)

	// 4. Unexpected error -> 500
	w4 := httptest.NewRecorder()
	c4, _ := gin.CreateTestContext(w4)
	c4.Set(authutils.ContextClaimsKey, validClaims)
	ok4 := authutils.CheckCourseAccess(c4, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return false, errors.New("db error")
	}, notFoundErr)
	assert.False(suite.T(), ok4)
	assert.Equal(suite.T(), http.StatusInternalServerError, w4.Code)

	// 5. Forbidden (!allowed) -> 403
	w5 := httptest.NewRecorder()
	c5, _ := gin.CreateTestContext(w5)
	c5.Set(authutils.ContextClaimsKey, validClaims)
	ok5 := authutils.CheckCourseAccess(c5, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return false, nil
	}, notFoundErr)
	assert.False(suite.T(), ok5)
	assert.Equal(suite.T(), http.StatusForbidden, w5.Code)

	// 6. Allowed -> true
	w6 := httptest.NewRecorder()
	c6, _ := gin.CreateTestContext(w6)
	c6.Set(authutils.ContextClaimsKey, validClaims)
	ok6 := authutils.CheckCourseAccess(c6, courseUuid, func(cUuid, uUuid uuid.UUID, isAdmin bool) (bool, error) {
		return true, nil
	}, notFoundErr)
	assert.True(suite.T(), ok6)
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
