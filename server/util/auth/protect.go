package auth

import (
	"net/http"
	"slices"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ContextClaimsKey   = "claims"
	ContextUserUuidKey = "userUuid"
	ContextRoleKey     = "role"
)

// Protect protects routes allowing access only to given roles
// if roles are empty it only checks for the validity of tokens
func Protect(roles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Missing token")
			return
		}

		token, claims, err := ParseToken(authHeader)
		if err != nil {
			zap.S().Infof("Auth failed with err = %+v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Invalid token format")
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Invalid token")
			return
		}

		if len(roles) != 0 && !slices.Contains(roles, claims.Role) {
			c.AbortWithStatus(http.StatusForbidden)
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
