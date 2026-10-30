package submission

import (
	"time"

	"github.com/google/uuid"
)

// CreateSubmissionDto represents the request payload sent by a student when submitting an assignment.
type CreateSubmissionDto struct {
	PayloadType PayloadType `json:"payloadType" binding:"required,oneof=URL FILE" example:"URL"`
	PayloadURL  string      `json:"payloadUrl" binding:"required,url,max=2048" example:"https://git.app.uib.no/student/exercise1.git"`
}

// SubmissionAcceptedDto is the immediate 202 response confirming the submission has been queued.
type SubmissionAcceptedDto struct {
	Uuid        uuid.UUID `json:"uuid" example:"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"`
	SubmittedAt time.Time `json:"submittedAt"`
	Status      string    `json:"status" example:"QUEUED"`
}
