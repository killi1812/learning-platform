package submission_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockAcknowledger struct {
	acked       bool
	nacked      bool
	requeue     bool
	ackErr      error
	nackErr     error
	deliveryTag uint64
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.acked = true
	m.deliveryTag = tag
	return m.ackErr
}

func (m *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	m.nacked = true
	m.requeue = requeue
	m.deliveryTag = tag
	return m.nackErr
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

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

		msgNilUuid := submission.SubmissionMessage{
			Uuid:        uuid.Nil,
			ContentID:   1,
			UserID:      1,
			PayloadURL:  "https://example.com",
			PayloadType: submission.PayloadTypeURL,
		}
		dataNil, _ := json.Marshal(msgNilUuid)
		errNil := consumer.Process(dataNil)
		assert.ErrorIs(t, errNil, submission.ErrInvalidMessage)
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

func TestSubmissionConsumer_RunAndHandle(t *testing.T) {
	logger := zap.NewNop().Sugar()

	validMsg := submission.SubmissionMessage{
		Uuid:        uuid.New(),
		ContentID:   1,
		UserID:      1,
		UserEmail:   "student@example.com",
		PayloadType: submission.PayloadTypeURL,
		PayloadURL:  "https://git.app.uib.no/repo.git",
		SubmittedAt: time.Now(),
	}
	validBody, _ := json.Marshal(validMsg)

	t.Run("Acks valid delivery and handles ack error", func(t *testing.T) {
		ack1 := &mockAcknowledger{}
		ack2 := &mockAcknowledger{ackErr: errors.New("ack failed")}

		client := &mockRabbitClient{
			consumeFunc: func(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error {
				assert.Equal(t, submission.SubmissionQueue, queue)
				assert.Equal(t, 50, prefetch)
				handler(amqp.Delivery{Acknowledger: ack1, DeliveryTag: 1, Body: validBody})
				handler(amqp.Delivery{Acknowledger: ack2, DeliveryTag: 2, Body: validBody})
				return nil
			},
		}

		worker := submission.NewSubmissionConsumer(client, &mockSubmissionService{}, logger)
		worker.Run(context.Background())

		assert.True(t, ack1.acked)
		assert.True(t, ack2.acked)
	})

	t.Run("Dead-letters invalid message without requeue", func(t *testing.T) {
		ack := &mockAcknowledger{nackErr: errors.New("nack failed")}
		client := &mockRabbitClient{
			consumeFunc: func(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error {
				handler(amqp.Delivery{Acknowledger: ack, Body: []byte("bad json")})
				return nil
			},
		}

		worker := submission.NewSubmissionConsumer(client, &mockSubmissionService{}, logger)
		worker.Run(context.Background())

		assert.True(t, ack.nacked)
		assert.False(t, ack.requeue)
	})

	t.Run("Requeues transient failure once, then dead-letters on redelivery", func(t *testing.T) {
		firstTry := &mockAcknowledger{nackErr: errors.New("requeue err")}
		secondTry := &mockAcknowledger{nackErr: errors.New("dlq err")}

		client := &mockRabbitClient{
			consumeFunc: func(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error {
				handler(amqp.Delivery{Acknowledger: firstTry, Redelivered: false, Body: validBody})
				handler(amqp.Delivery{Acknowledger: secondTry, Redelivered: true, Body: validBody})
				return errors.New("consume loop terminated")
			},
		}

		mockSvc := &mockSubmissionService{
			saveFromMessageFunc: func(msg submission.SubmissionMessage) (*submission.IndividualSubmission, error) {
				return nil, errors.New("transient db timeout")
			},
		}

		worker := submission.NewSubmissionConsumer(client, mockSvc, logger)
		worker.Run(context.Background())

		assert.True(t, firstTry.nacked)
		assert.True(t, firstTry.requeue)

		assert.True(t, secondTry.nacked)
		assert.False(t, secondTry.requeue)
	})
}
