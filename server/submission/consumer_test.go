package submission_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSubmissionConsumer_Process(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("Valid message processes and calls SaveFromMessage", func(t *testing.T) {
		called := false
		subUUID := uuid.New()
		mockSvc := &mockSubmissionService{
			saveFromMessageFunc: func(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error) {
				called = true
				assert.Equal(t, subUUID, msg.Uuid)
				assert.Equal(t, uint(10), msg.ContentID)
				assert.Equal(t, uint(20), msg.UserID)
				return &submission.IndividualSubmission{
					Submission: submission.Submission{
						Uuid:    subUUID,
						Attempt: 1,
						IsLate:  false,
					},
				}, nil
			},
		}

		consumerWorker := submission.NewSubmissionConsumer(nil, mockSvc, logger)
		consumer, ok := consumerWorker.(*submission.SubmissionConsumer)
		assert.True(t, ok)

		msg := submission.SubmissionMessage{
			Uuid:        subUUID,
			ContentID:   10,
			UserID:      20,
			UserEmail:   "student@hvl.no",
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/student/task.git",
			SubmittedAt: time.Now(),
		}
		data, _ := json.Marshal(msg)

		err := consumer.Process(data)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Malformed JSON returns ErrInvalidMessage", func(t *testing.T) {
		consumerWorker := submission.NewSubmissionConsumer(nil, &mockSubmissionService{}, logger)
		consumer := consumerWorker.(*submission.SubmissionConsumer)

		err := consumer.Process([]byte("not valid json"))
		assert.ErrorIs(t, err, submission.ErrInvalidMessage)
	})

	t.Run("Missing required fields returns ErrInvalidMessage", func(t *testing.T) {
		consumerWorker := submission.NewSubmissionConsumer(nil, &mockSubmissionService{}, logger)
		consumer := consumerWorker.(*submission.SubmissionConsumer)

		msg := submission.SubmissionMessage{
			Uuid:      uuid.New(),
			ContentID: 0, // missing
			UserID:    1,
		}
		data, _ := json.Marshal(msg)

		err := consumer.Process(data)
		assert.ErrorIs(t, err, submission.ErrInvalidMessage)
	})

	t.Run("Service failure returns error", func(t *testing.T) {
		mockSvc := &mockSubmissionService{
			saveFromMessageFunc: func(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error) {
				return nil, errors.New("db connection failure")
			},
		}
		consumerWorker := submission.NewSubmissionConsumer(nil, mockSvc, logger)
		consumer := consumerWorker.(*submission.SubmissionConsumer)

		msg := submission.SubmissionMessage{
			Uuid:        uuid.New(),
			ContentID:   1,
			UserID:      1,
			UserEmail:   "student@hvl.no",
			PayloadType: submission.PayloadTypeURL,
			PayloadURL:  "https://git.app.uib.no/repo.git",
			SubmittedAt: time.Now(),
		}
		data, _ := json.Marshal(msg)

		err := consumer.Process(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db connection failure")
	})
}
