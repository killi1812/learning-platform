package notification

import (
	"errors"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrRecipientNotFound    = errors.New("one or more recipients not found")
)

// sendBatchSize limits rows per INSERT so large recipient lists stay within Postgres parameter limits.
const sendBatchSize = 100

type INotificationService interface {
	SendToUsers(senderUuid uuid.UUID, recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error)
	NotifySystem(recipientUuid uuid.UUID, content NotificationContent) (*Notification, error)
	NotifySystemMany(recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error)
	List(userUuid uuid.UUID, limit int, before *time.Time) ([]Notification, error)
	CountUnseen(userUuid uuid.UUID) (int64, error)
	MarkSeen(userUuid uuid.UUID, notificationUuid uuid.UUID) (*Notification, error)
	MarkAllSeen(userUuid uuid.UUID) (int64, error)
	Delete(userUuid uuid.UUID, notificationUuid uuid.UUID) error
}

type NotificationService struct {
	db     *gorm.DB
	hub    *Hub
	logger *zap.SugaredLogger
}

func NewNotificationService(db *gorm.DB, hub *Hub, logger *zap.SugaredLogger) INotificationService {
	return &NotificationService{
		db:     db,
		hub:    hub,
		logger: logger,
	}
}

func (s *NotificationService) userID(userUuid uuid.UUID) (uint, error) {
	var u user.User
	if err := s.db.Select("id").Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	return u.ID, nil
}

// NotifySystem writes one row with no sender for a single recipient, for example a welcome message.
// The row is pushed to the recipient's open streams, like SendToUsers.
func (s *NotificationService) NotifySystem(recipientUuid uuid.UUID, content NotificationContent) (*Notification, error) {
	userID, err := s.userID(recipientUuid)
	if err != nil {
		return nil, err
	}

	n := Notification{
		UserID:    userID,
		Title:     content.Title,
		Body:      content.Body,
		URL:       content.URL,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&n).Error; err != nil {
		s.logger.Errorf("Failed to create system notification: %v", err)
		return nil, err
	}

	s.hub.Publish(recipientUuid, NotificationDto{}.FromModel(&n))
	return &n, nil
}

// SendToUsers writes one private row per recipient and pushes each row to that user's open streams.
// Rows are written in one transaction, so either every recipient gets the notification or none do.
func (s *NotificationService) SendToUsers(senderUuid uuid.UUID, recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error) {
	senderID, err := s.userID(senderUuid)
	if err != nil {
		return nil, err
	}
	return s.sendBatch(&senderID, recipientUuids, content)
}

// NotifySystemMany writes system notifications (SenderID = nil) to multiple recipients in batches.
func (s *NotificationService) NotifySystemMany(recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error) {
	return s.sendBatch(nil, recipientUuids, content)
}

func (s *NotificationService) sendBatch(senderID *uint, recipientUuids []uuid.UUID, content NotificationContent) ([]Notification, error) {
	if len(recipientUuids) == 0 {
		return []Notification{}, nil
	}

	unique := make(map[uuid.UUID]struct{}, len(recipientUuids))
	for _, id := range recipientUuids {
		unique[id] = struct{}{}
	}
	uuids := make([]uuid.UUID, 0, len(unique))
	for id := range unique {
		uuids = append(uuids, id)
	}

	var recipients []user.User
	if err := s.db.Select("id", "uuid").Where("uuid IN ?", uuids).Find(&recipients).Error; err != nil {
		return nil, err
	}
	if len(recipients) != len(uuids) {
		return nil, ErrRecipientNotFound
	}

	now := time.Now()
	rows := make([]Notification, 0, len(recipients))
	uuidByID := make(map[uint]uuid.UUID, len(recipients))
	for _, r := range recipients {
		uuidByID[r.ID] = r.Uuid
		rows = append(rows, Notification{
			UserID:    r.ID,
			SenderID:  senderID,
			Title:     content.Title,
			Body:      content.Body,
			URL:       content.URL,
			CreatedAt: now,
		})
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(&rows, sendBatchSize).Error
	})
	if err != nil {
		s.logger.Errorf("Failed to send notifications: %v", err)
		return nil, err
	}

	for i := range rows {
		s.hub.Publish(uuidByID[rows[i].UserID], NotificationDto{}.FromModel(&rows[i]))
	}
	return rows, nil
}

// List returns the caller's own notifications, newest first.
// before is an exclusive created_at cursor for paging.
func (s *NotificationService) List(userUuid uuid.UUID, limit int, before *time.Time) ([]Notification, error) {
	userID, err := s.userID(userUuid)
	if err != nil {
		return nil, err
	}

	q := s.db.Where("user_id = ?", userID)
	if before != nil {
		q = q.Where("created_at < ?", *before)
	}

	var out []Notification
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NotificationService) CountUnseen(userUuid uuid.UUID) (int64, error) {
	userID, err := s.userID(userUuid)
	if err != nil {
		return 0, err
	}

	var count int64
	err = s.db.Model(&Notification{}).
		Where("user_id = ? AND seen_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

// MarkSeen sets seen_at once. Repeat calls keep the first timestamp.
func (s *NotificationService) MarkSeen(userUuid uuid.UUID, notificationUuid uuid.UUID) (*Notification, error) {
	userID, err := s.userID(userUuid)
	if err != nil {
		return nil, err
	}

	var n Notification
	if err := s.db.Where("uuid = ? AND user_id = ?", notificationUuid, userID).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}

	if n.SeenAt == nil {
		now := time.Now()
		if err := s.db.Model(&n).Update("seen_at", now).Error; err != nil {
			return nil, err
		}
		n.SeenAt = &now
	}
	return &n, nil
}

func (s *NotificationService) MarkAllSeen(userUuid uuid.UUID) (int64, error) {
	userID, err := s.userID(userUuid)
	if err != nil {
		return 0, err
	}

	res := s.db.Model(&Notification{}).
		Where("user_id = ? AND seen_at IS NULL", userID).
		Update("seen_at", time.Now())
	return res.RowsAffected, res.Error
}

// Delete hard-deletes the caller's own row. Other users' rows are not touched.
func (s *NotificationService) Delete(userUuid uuid.UUID, notificationUuid uuid.UUID) error {
	userID, err := s.userID(userUuid)
	if err != nil {
		return err
	}

	res := s.db.Where("uuid = ? AND user_id = ?", notificationUuid, userID).Delete(&Notification{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}
