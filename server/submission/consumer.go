package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/rabbitmq"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var (
	ErrInvalidMessage = errors.New("invalid submission message payload")
)

// SubmissionConsumer consumes submission events from RabbitMQ and inserts them into PostgreSQL.
type SubmissionConsumer struct {
	client  rabbitmq.IClient
	service ISubmissionService
	logger  *zap.SugaredLogger
}

// NewSubmissionConsumer constructs a new SubmissionConsumer worker.
func NewSubmissionConsumer(client rabbitmq.IClient, service ISubmissionService, logger *zap.SugaredLogger) app.Worker {
	return &SubmissionConsumer{
		client:  client,
		service: service,
		logger:  logger,
	}
}

// Run starts the consumer loop until ctx is canceled.
func (c *SubmissionConsumer) Run(ctx context.Context) {
	c.logger.Infof("Starting submission consumer worker on queue %s", SubmissionQueue)

	err := c.client.Consume(ctx, SubmissionQueue, 50, c.handle)
	if err != nil && !errors.Is(err, context.Canceled) {
		c.logger.Errorf("Submission consumer exited with error: %v", err)
	} else {
		c.logger.Infof("Submission consumer stopped")
	}
}

func (c *SubmissionConsumer) handle(d amqp.Delivery) {
	err := c.Process(d.Body)
	if err == nil {
		if ackErr := d.Ack(false); ackErr != nil {
			c.logger.Errorf("Failed to ack delivery: %v", ackErr)
		}
		return
	}

	if errors.Is(err, ErrInvalidMessage) {
		c.logger.Warnf("Dead-lettering invalid submission message: %v", err)
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Errorf("Failed to nack invalid delivery to DLQ: %v", nackErr)
		}
		return
	}

	// Database or transient persistence failure
	if !d.Redelivered {
		c.logger.Warnf("Transient failure processing submission; requeuing once: %v", err)
		time.Sleep(200 * time.Millisecond)
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Errorf("Failed to requeue delivery: %v", nackErr)
		}
	} else {
		c.logger.Errorf("Persistent failure processing submission; sending to DLQ: %v", err)
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Errorf("Failed to dead-letter delivery: %v", nackErr)
		}
	}
}

// Process parses and persists the submission message.
func (c *SubmissionConsumer) Process(body []byte) error {
	var msg SubmissionMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("%w: failed to parse JSON: %v", ErrInvalidMessage, err)
	}

	if msg.Uuid == uuid.Nil || msg.ContentID == 0 || msg.UserID == 0 || msg.PayloadURL == "" {
		return fmt.Errorf("%w: missing required fields", ErrInvalidMessage)
	}

	saved, err := c.service.SaveFromMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to save submission: %w", err)
	}

	latency := time.Since(msg.SubmittedAt)
	c.logger.Debugf("Submission %s persisted successfully (attempt=%d, isLate=%t, latency=%v)",
		saved.Uuid, saved.Attempt, saved.IsLate, latency)

	return nil
}
