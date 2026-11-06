package ginutil

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrorResponse represents a standardized API error response payload.
type ErrorResponse struct {
	Error string `json:"error" example:"Example error, something went wrong"`
}

// RespondWithError halts the Gin context and writes a standardized JSON error response.
func RespondWithError(c *gin.Context, statusCode int, message string) {
	c.AbortWithStatusJSON(statusCode, ErrorResponse{Error: message})
}

// ParseUUIDParam extracts and parses a UUID route parameter, aborting with 400 Bad Request on failure.
func ParseUUIDParam(c *gin.Context, param string) (uuid.UUID, bool) {
	parsedUUID, err := uuid.Parse(c.Param(param))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid UUID parameter: "+param)
		return uuid.Nil, false
	}
	return parsedUUID, true
}

// HandleServiceError inspects a service error.
// If err is nil, returns false.
// If err matches notFoundErr, aborts with 404 Not Found and returns true.
// Otherwise, aborts with 500 Internal Server Error and returns true.
func HandleServiceError(c *gin.Context, err error, notFoundErr error) bool {
	zap.S().Debugf("Inside HandleServiceError")
	if err == nil {
		zap.S().Debugf("err was nil")
		return false
	}
	zap.S().Debugf("err was not nil")
	if notFoundErr != nil && errors.Is(err, notFoundErr) {
		zap.S().Debugf("Inside if statement")
		RespondWithError(c, http.StatusNotFound, err.Error())
		return true
	}
	zap.S().Debugf("not inside if statement")
	RespondWithError(c, http.StatusInternalServerError, "Internal server error")
	return true
}
