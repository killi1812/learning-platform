package submission

import (
	"time"

	"github.com/google/uuid"
)

// SubmissionQueue is queue name
const SubmissionQueue = "submission.created"

// SubmissionMessage is the RabbitMQ message payload for an individual submission.
type SubmissionMessage struct {
	Uuid        uuid.UUID   `json:"uuid"`
	ContentID   uint        `json:"contentId"`
	UserID      uint        `json:"userId"`
	UserEmail   string      `json:"userEmail"`
	PayloadType PayloadType `json:"payloadType"`
	PayloadURL  string      `json:"payloadUrl"`
	SubmittedAt time.Time   `json:"submittedAt"`
}
