package notification

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type INotifier interface {
	NotifySystem(recipientUuid uuid.UUID, content NotificationContent) (*Notification, error)
	NotifySystemMany(recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error)
}

type Notifier struct {
	db     *gorm.DB
	hub    *Hub
	logger *zap.SugaredLogger
}

func NewNotifier(db *gorm.DB, hub *Hub, logger *zap.SugaredLogger) INotifier {
	return &Notifier{
		db:     db,
		hub:    hub,
		logger: logger,
	}
}

// NotifySystem writes one row with no sender for a single recipient, for example a welcome message.
// The row is pushed to the recipient's open streams, like SendToUsers.
func (n *Notifier) NotifySystem(recipientUuid uuid.UUID, content NotificationContent) (*Notification, error) {
	userID, err := lookupUserID(n.db, recipientUuid)
	if err != nil {
		return nil, err
	}

	notif := Notification{
		UserID:    userID,
		Title:     content.Title,
		Body:      content.Body,
		URL:       content.URL,
		CreatedAt: time.Now(),
	}
	if err := n.db.Create(&notif).Error; err != nil {
		n.logger.Errorf("Failed to create system notification: %v", err)
		return nil, err
	}

	n.hub.Publish(recipientUuid, NotificationDto{}.FromModel(&notif))
	return &notif, nil
}

// NotifySystemMany writes system notifications (SenderID = nil) to multiple recipients in batches.
func (n *Notifier) NotifySystemMany(recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error) {
	return sendBatch(n.db, n.hub, n.logger, nil, recipientUuids, content)
}
