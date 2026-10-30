package authutils

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ContextClaimsKey   = "claims"
	ContextUserUuidKey = "userUuid"
	ContextRoleKey     = "role"
)

// SessionValidator checks whether a session is still active and valid in the database.
type SessionValidator func(ctx context.Context, userUuid, sessionUuid uuid.UUID) bool

// CourseAccessChecker checks whether a user (or system admin) has required access to a course.
type CourseAccessChecker func(courseUuid, userUuid uuid.UUID, isSysAdmin bool) (bool, error)

var (
	sessionValidatorMu     sync.RWMutex
	globalSessionValidator SessionValidator
)

// SetSessionValidator registers the active session validator callback.
func SetSessionValidator(validator SessionValidator) {
	sessionValidatorMu.Lock()
	defer sessionValidatorMu.Unlock()
	globalSessionValidator = validator
}

func getSessionValidator() SessionValidator {
	sessionValidatorMu.RLock()
	defer sessionValidatorMu.RUnlock()
	return globalSessionValidator
}

// Protect protects routes allowing access only to given roles
// if roles are empty it only checks for the validity of tokens
func Protect(roles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ginutil.RespondWithError(c, http.StatusUnauthorized, "Missing token")
			return
		}

		token, claims, err := ParseToken(authHeader)
		if err != nil {
			zap.S().Infof("Auth failed with err = %+v", err)
			ginutil.RespondWithError(c, http.StatusUnauthorized, "Invalid token format")
			return
		}

		if !token.Valid {
			ginutil.RespondWithError(c, http.StatusUnauthorized, "Invalid token")
			return
		}

		validator := getSessionValidator()
		if validator != nil && claims.SessionUuid != uuid.Nil {
			userUuid, err := uuid.Parse(claims.ID)
			if err != nil || !validator(c.Request.Context(), userUuid, claims.SessionUuid) {
				ginutil.RespondWithError(c, http.StatusUnauthorized, "Session revoked or expired")
				return
			}
		}

		if len(roles) != 0 && !slices.Contains(roles, claims.Role) {
			ginutil.RespondWithError(c, http.StatusForbidden, "Forbidden: insufficient permissions")
			return
		}

		c.Set(ContextClaimsKey, claims)
		c.Set(ContextUserUuidKey, claims.ID)
		c.Set(ContextRoleKey, claims.Role)

		c.Next()
	}
}

// GetClaims retrieves parsed JWT claims from gin context
func GetClaims(c *gin.Context) (*Claims, bool) {
	val, exists := c.Get(ContextClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*Claims)
	return claims, ok
}

// GetUserUUID retrieves user UUID from claims in gin context
func GetUserUUID(c *gin.Context) (uuid.UUID, error) {
	claims, ok := GetClaims(c)
	if !ok || claims == nil {
		return uuid.Nil, cerror.ErrUserIsNil
	}
	return uuid.Parse(claims.ID)
}

// GetUserRole retrieves user role from claims in gin context
func GetUserRole(c *gin.Context) (Role, error) {
	claims, ok := GetClaims(c)
	if !ok || claims == nil {
		return "", cerror.ErrUserIsNil
	}
	return claims.Role, nil
}

// IsAdmin returns true if the authenticated user has ROLE_ADMIN.
func IsAdmin(c *gin.Context) bool {
	claims, ok := GetClaims(c)
	return ok && claims != nil && claims.Role == ROLE_ADMIN
}

// CheckCourseAccess verifies the authenticated user's access to a course using the provided checker.
// Aborts with 401/403/404/500 and returns false if access is denied or an error occurs.
func CheckCourseAccess(c *gin.Context, courseUuid uuid.UUID, checker CourseAccessChecker, notFoundErr error) bool {
	claims, ok := GetClaims(c)
	if !ok || claims == nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return false
	}
	userUuid, err := GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	isSysAdmin := claims.Role == ROLE_ADMIN
	allowed, err := checker(courseUuid, userUuid, isSysAdmin)
	if err != nil {
		if notFoundErr != nil && errors.Is(err, notFoundErr) {
			ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
			return false
		}
		zap.S().Errorf("Failed to check course role: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Internal server error")
		return false
	}
	if !allowed {
		ginutil.RespondWithError(c, http.StatusForbidden, "Forbidden: insufficient permissions")
		return false
	}
	return true
}
