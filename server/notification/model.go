package notification

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification is a private inbox entry. Each recipient gets their own row.
// Content is immutable: no UpdatedAt and no soft delete. Only SeenAt changes,
// and Delete removes the row. To correct a sent notification, send a new one.
type Notification struct {
	ID        uint       `gorm:"primaryKey"`
	Uuid      uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	UserID    uint       `gorm:"not null;index:idx_notification_user_created"`
	SenderID  *uint      `gorm:"index"`
	Title     string     `gorm:"type:varchar(255);not null"`
	Body      string     `gorm:"type:text;not null"`
	URL       string     `gorm:"type:varchar(2048);not null;default:''"`
	CreatedAt time.Time  `gorm:"not null;index:idx_notification_user_created"`
	SeenAt    *time.Time `gorm:"index"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.Uuid == uuid.Nil {
		n.Uuid = uuid.New()
	}
	return nil
}
