package submission

import (
	"time"

	"github.com/google/uuid"
)

// CreateSubmissionDto represents the request payload sent by a student when submitting an assignment.
type CreateSubmissionDto struct {
	PayloadType PayloadType `json:"payloadType" binding:"required,oneof=URL FILE" example:"URL"`
	PayloadURL  string      `json:"payloadUrl" binding:"required,http_url,max=2048" example:"https://git.app.uib.no/student/exercise1.git"`
}

// SubmissionStatus represents the queuing or processing state of an accepted submission.
type SubmissionStatus string

const (
	SubmissionStatusQueued SubmissionStatus = "QUEUED"
)

// SubmissionAcceptedDto is the immediate 202 response confirming the submission has been queued.
type SubmissionAcceptedDto struct {
	Uuid        uuid.UUID        `json:"uuid" example:"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"`
	SubmittedAt time.Time        `json:"submittedAt"`
	Status      SubmissionStatus `json:"status" example:"QUEUED"`
}

// UploadTicketDto is returned when initiating a two-phase file submission.
type UploadTicketDto struct {
	TicketUuid  uuid.UUID `json:"ticketUuid"`
	SubmittedAt time.Time `json:"submittedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// SubmissionDto represents an individual or group submission returned in student/teacher listings.
type SubmissionDto struct {
	Uuid         uuid.UUID   `json:"uuid"`
	Attempt      int         `json:"attempt"`
	IsLate       bool        `json:"isLate"`
	PayloadType  PayloadType `json:"payloadType"`
	PayloadURL   string      `json:"payloadUrl,omitempty"`
	FileName     string      `json:"fileName,omitempty"`
	FileSize     int64       `json:"fileSize,omitempty"`
	ContentType  string      `json:"contentType,omitempty"`
	SubmittedAt  time.Time   `json:"submittedAt"`
	UserEmail    string      `json:"userEmail,omitempty"`
	IsGroup      bool        `json:"isGroup"`
	GroupName    string      `json:"groupName,omitempty"`
	MemberEmails []string    `json:"memberEmails,omitempty"`
}
