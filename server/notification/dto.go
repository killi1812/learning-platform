package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationDto struct {
	Uuid      uuid.UUID  `json:"uuid"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"createdAt"`
	SeenAt    *time.Time `json:"seenAt"`
}

func (NotificationDto) FromModel(n *Notification) NotificationDto {
	return NotificationDto{
		Uuid:      n.Uuid,
		Title:     n.Title,
		Body:      n.Body,
		URL:       n.URL,
		CreatedAt: n.CreatedAt,
		SeenAt:    n.SeenAt,
	}
}

type SendNotificationDto struct {
	UserUuids []uuid.UUID `json:"userUuids" binding:"required,min=1,max=500"`
	Title     string      `json:"title" binding:"required,min=1,max=255"`
	Body      string      `json:"body" binding:"required,min=1,max=5000"`
	// URL is an internal path such as /courses/<uuid>. Absolute and protocol-relative URLs are rejected.
	URL string `json:"url" binding:"omitempty,max=2048,startswith=/,excludes=//"`
}

type UnreadCountDto struct {
	Count int64 `json:"count"`
}

type MarkAllSeenDto struct {
	Updated int64 `json:"updated"`
}
