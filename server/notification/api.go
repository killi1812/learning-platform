package notification

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	// heartbeatInterval keeps idle SSE connections alive through proxies.
	heartbeatInterval = 25 * time.Second
)

type NotificationCtn struct {
	service INotificationService
	hub     *Hub
	logger  *zap.SugaredLogger
}

func NewNotificationCtn(service INotificationService, hub *Hub, logger *zap.SugaredLogger) app.Api {
	return &NotificationCtn{
		service: service,
		hub:     hub,
		logger:  logger,
	}
}

func (ctn *NotificationCtn) RegisterEndpoints(api *gin.RouterGroup) {
	notifications := api.Group("/notification")

	notifications.GET("", authutils.Protect(), ctn.list)
	notifications.GET("/unread-count", authutils.Protect(), ctn.unreadCount)
	notifications.GET("/stream", authutils.Protect(), ctn.stream)
	notifications.POST("/seen-all", authutils.Protect(), ctn.markAllSeen)
	notifications.POST("/:uuid/seen", authutils.Protect(), ctn.markSeen)
	notifications.DELETE("/:uuid", authutils.Protect(), ctn.delete)

	notifications.POST("/send", authutils.Protect(), ctn.send)
}

func (ctn *NotificationCtn) handleError(c *gin.Context, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotificationNotFound),
		errors.Is(err, ErrRecipientNotFound):
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrUserNotFound):
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, fallbackMsg)
	}
	return true
}

// ListNotifications godoc
//
//	@Summary		List own notifications
//	@Description	Returns the caller's notifications, newest first. Use before (RFC3339 created_at) to page.
//	@Tags			notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int		false	"Page size (max 100)"
//	@Param			before	query		string	false	"Return notifications created before this RFC3339 time"
//	@Success		200		{array}		NotificationDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Router			/notification [get]
func (ctn *NotificationCtn) list(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit := defaultPageSize
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid limit")
			return
		}
		limit = min(parsed, maxPageSize)
	}

	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid before timestamp")
			return
		}
		before = &parsed
	}

	items, err := ctn.service.List(userUuid, limit, before)
	if ctn.handleError(c, err, "Failed to list notifications") {
		return
	}

	dtos := make([]NotificationDto, 0, len(items))
	for i := range items {
		dtos = append(dtos, NotificationDto{}.FromModel(&items[i]))
	}
	c.JSON(http.StatusOK, dtos)
}

// UnreadCount godoc
//
//	@Summary	Count unseen notifications
//	@Tags		notification
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	UnreadCountDto
//	@Failure	401	{object}	ginutil.ErrorResponse
//	@Router		/notification/unread-count [get]
func (ctn *NotificationCtn) unreadCount(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	count, err := ctn.service.CountUnseen(userUuid)
	if ctn.handleError(c, err, "Failed to count notifications") {
		return
	}
	c.JSON(http.StatusOK, UnreadCountDto{Count: count})
}

// StreamNotifications godoc
//
//	@Summary		Stream new notifications (SSE)
//	@Description	Server-Sent Events stream. Emits "notification" events with NotificationDto payload. Closes when the access token expires; client must reconnect with a fresh token and refetch the inbox.
//	@Tags			notification
//	@Produce		text/event-stream
//	@Security		BearerAuth
//	@Success		200	{string}	string	"event stream"
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Router			/notification/stream [get]
func (ctn *NotificationCtn) stream(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	claims, ok := authutils.GetClaims(c)
	if !ok || claims.ExpiresAt == nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// The server WriteTimeout (app/httpServer.go) would cut a long-lived stream. Clear the deadline for this request only.
	// Ignored error: writers without deadline support keep the server default.
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	// Send headers now. Gin only flushes after the first step, which blocks until the next event or heartbeat.
	c.Writer.WriteHeaderNow()
	c.Writer.Flush()

	events, unsubscribe := ctn.hub.Subscribe(userUuid)
	defer unsubscribe()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	expiry := time.NewTimer(time.Until(claims.ExpiresAt.Time))
	defer expiry.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case <-expiry.C:
			return false
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			return true
		case dto := <-events:
			c.SSEvent("notification", dto)
			return true
		}
	})
}

// MarkNotificationSeen godoc
//
//	@Summary	Mark notification as seen
//	@Tags		notification
//	@Produce	json
//	@Security	BearerAuth
//	@Param		uuid	path		string	true	"Notification UUID"
//	@Success	200		{object}	NotificationDto
//	@Failure	400		{object}	ginutil.ErrorResponse
//	@Failure	404		{object}	ginutil.ErrorResponse
//	@Router		/notification/{uuid}/seen [post]
func (ctn *NotificationCtn) markSeen(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	notificationUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	n, err := ctn.service.MarkSeen(userUuid, notificationUuid)
	if ctn.handleError(c, err, "Failed to mark notification as seen") {
		return
	}
	c.JSON(http.StatusOK, NotificationDto{}.FromModel(n))
}

// MarkAllSeen godoc
//
//	@Summary	Mark all notifications as seen
//	@Tags		notification
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	MarkAllSeenDto
//	@Failure	401	{object}	ginutil.ErrorResponse
//	@Router		/notification/seen-all [post]
func (ctn *NotificationCtn) markAllSeen(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	updated, err := ctn.service.MarkAllSeen(userUuid)
	if ctn.handleError(c, err, "Failed to mark notifications as seen") {
		return
	}
	c.JSON(http.StatusOK, MarkAllSeenDto{Updated: updated})
}

// DeleteNotification godoc
//
//	@Summary	Delete own notification
//	@Tags		notification
//	@Security	BearerAuth
//	@Param		uuid	path	string	true	"Notification UUID"
//	@Success	204
//	@Failure	400	{object}	ginutil.ErrorResponse
//	@Failure	404	{object}	ginutil.ErrorResponse
//	@Router		/notification/{uuid} [delete]
func (ctn *NotificationCtn) delete(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	notificationUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if ctn.handleError(c, ctn.service.Delete(userUuid, notificationUuid), "Failed to delete notification") {
		return
	}
	c.Status(http.StatusNoContent)
}

// SendNotification godoc
//
//	@Summary		Send notification to users
//	@Description	Creates one private notification per recipient and pushes to their open streams. Admin only.
//	@Tags			notification
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			notification	body		SendNotificationDto	true	"Recipients and content"
//	@Success		201				{array}		NotificationDto
//	@Failure		400				{object}	ginutil.ErrorResponse
//	@Failure		401				{object}	ginutil.ErrorResponse
//	@Failure		403				{object}	ginutil.ErrorResponse
//	@Failure		404				{object}	ginutil.ErrorResponse
//	@Router			/notification/send [post]
func (ctn *NotificationCtn) send(c *gin.Context) {
	senderUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input SendNotificationDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	sent, err := ctn.service.SendToUsers(senderUuid, input.UserUuids, input.Title, input.Body, input.URL)
	if ctn.handleError(c, err, "Failed to send notifications") {
		return
	}

	dtos := make([]NotificationDto, 0, len(sent))
	for i := range sent {
		dtos = append(dtos, NotificationDto{}.FromModel(&sent[i]))
	}
	c.JSON(http.StatusCreated, dtos)
}
