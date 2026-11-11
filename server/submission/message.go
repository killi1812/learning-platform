package submission

import (
	"time"

	"github.com/google/uuid"
)

// SubmissionQueue is queue name
const SubmissionQueue = "submission.created"

// SubmissionMessage is the RabbitMQ message payload for an individual submission.
type SubmissionMessage struct {
	Uuid         uuid.UUID   `json:"uuid"`
	ContentID    uint        `json:"contentId"`
	ContentTitle string      `json:"contentTitle,omitempty"`
	CourseUuid   uuid.UUID   `json:"courseUuid,omitempty"`
	UserID       uint        `json:"userId"`
	UserUuid     uuid.UUID   `json:"userUuid,omitempty"`
	UserEmail    string      `json:"userEmail"`
	PayloadType  PayloadType `json:"payloadType"`
	PayloadURL   string      `json:"payloadUrl,omitempty"`
	FileName     string      `json:"fileName,omitempty"`
	FileSize     int64       `json:"fileSize,omitempty"`
	ContentType  string      `json:"contentType,omitempty"`
	StorageKey   string      `json:"storageKey,omitempty"`
	SubmittedAt  time.Time   `json:"submittedAt"`
}
