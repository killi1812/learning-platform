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
	"github.com/stretchr/testify/require"
)

type mockRabbitClient struct {
	declareQueueFunc func(name string) error
	publishFunc      func(ctx context.Context, queue string, body []byte) error
	consumeFunc      func(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error
	closeFunc        func() error
}

func (m *mockRabbitClient) DeclareQueue(name string) error {
	if m.declareQueueFunc != nil {
		return m.declareQueueFunc(name)
	}
	return nil
}

func (m *mockRabbitClient) Publish(ctx context.Context, queue string, body []byte) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, queue, body)
	}
	return nil
}

func (m *mockRabbitClient) Consume(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, queue, prefetch, handler)
	}
	return nil
}

func (m *mockRabbitClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestSubmissionPublisher(t *testing.T) {
	t.Run("NewSubmissionPublisher declares submission queue and publishes message", func(t *testing.T) {
		var declaredQueue string
		var publishedQueue string
		var publishedBody []byte

		client := &mockRabbitClient{
			declareQueueFunc: func(name string) error {
				declaredQueue = name
				return nil
			},
			publishFunc: func(ctx context.Context, queue string, body []byte) error {
				publishedQueue = queue
				publishedBody = body
				return nil
			},
		}

		pub, err := submission.NewSubmissionPublisher(client)
		require.NoError(t, err)
		require.NotNil(t, pub)
		assert.Equal(t, submission.SubmissionQueue, declaredQueue)

		userUuid := uuid.New()
		courseUuid := uuid.New()
		msg := submission.SubmissionMessage{
			Uuid:         uuid.New(),
			ContentID:    5,
			ContentTitle: "Homework 1",
			CourseUuid:   courseUuid,
			UserID:       12,
			UserUuid:     userUuid,
			UserEmail:    "student@example.com",
			PayloadType:  submission.PayloadTypeURL,
			PayloadURL:   "https://git.app.uib.no/student/repo.git",
			SubmittedAt:  time.Now().UTC().Truncate(time.Second),
		}

		err = pub.Publish(context.Background(), msg)
		require.NoError(t, err)
		assert.Equal(t, submission.SubmissionQueue, publishedQueue)

		var decoded submission.SubmissionMessage
		require.NoError(t, json.Unmarshal(publishedBody, &decoded))
		assert.Equal(t, msg.Uuid, decoded.Uuid)
		assert.Equal(t, msg.ContentID, decoded.ContentID)
		assert.Equal(t, msg.ContentTitle, decoded.ContentTitle)
		assert.Equal(t, msg.CourseUuid, decoded.CourseUuid)
		assert.Equal(t, msg.UserID, decoded.UserID)
		assert.Equal(t, msg.UserUuid, decoded.UserUuid)
		assert.Equal(t, msg.PayloadURL, decoded.PayloadURL)
	})

	t.Run("NewSubmissionPublisher returns error when queue declaration fails", func(t *testing.T) {
		client := &mockRabbitClient{
			declareQueueFunc: func(name string) error {
				return errors.New("declare failed")
			},
		}

		pub, err := submission.NewSubmissionPublisher(client)
		assert.Error(t, err)
		assert.Nil(t, pub)
		assert.Contains(t, err.Error(), "failed to declare submission queue")
	})

	t.Run("Publish returns error when client publish fails", func(t *testing.T) {
		client := &mockRabbitClient{
			publishFunc: func(ctx context.Context, queue string, body []byte) error {
				return errors.New("broker unavailable")
			},
		}

		pub, err := submission.NewSubmissionPublisher(client)
		require.NoError(t, err)

		err = pub.Publish(context.Background(), submission.SubmissionMessage{
			Uuid:       uuid.New(),
			ContentID:  1,
			UserID:     1,
			PayloadURL: "https://git.app.uib.no/repo.git",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broker unavailable")
	})
}
