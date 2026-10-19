package ginutil

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorResponse represents a standardized API error response payload.
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid email or password"`
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

// ParseUintParam extracts and parses an unsigned integer route parameter, aborting with 400 Bad Request on failure.
func ParseUintParam(c *gin.Context, param string) (uint, bool) {
	valStr := c.Param(param)
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid uint parameter: "+param)
		return 0, false
	}
	return uint(val), true
}

// HandleServiceError inspects a service error.
// If err is nil, returns false.
// If err matches notFoundErr, aborts with 404 Not Found and returns true.
// Otherwise, aborts with 500 Internal Server Error and returns true.
func HandleServiceError(c *gin.Context, err error, notFoundErr error) bool {
	if err == nil {
		return false
	}
	if notFoundErr != nil && errors.Is(err, notFoundErr) {
		RespondWithError(c, http.StatusNotFound, err.Error())
		return true
	}
	RespondWithError(c, http.StatusInternalServerError, "Internal server error")
	return true
}
