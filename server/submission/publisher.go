package submission

import (
	"context"
	"encoding/json"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/rabbitmq"
)

// ISubmissionPublisher abstracts enqueuing submission events into RabbitMQ.
type ISubmissionPublisher interface {
	Publish(ctx context.Context, msg SubmissionMessage) error
}

// SubmissionPublisher publishes submission messages to RabbitMQ with publisher confirms.
type SubmissionPublisher struct {
	client *rabbitmq.Client
}

// NewSubmissionPublisher initializes queues and returns a new SubmissionPublisher.
func NewSubmissionPublisher(client *rabbitmq.Client) (ISubmissionPublisher, error) {
	if err := client.DeclareQueue(SubmissionQueue); err != nil {
		return nil, fmt.Errorf("failed to declare submission queue %s: %w", SubmissionQueue, err)
	}
	return &SubmissionPublisher{client: client}, nil
}

// Publish serializes and publishes the submission message to the submission queue.
func (p *SubmissionPublisher) Publish(ctx context.Context, msg SubmissionMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal submission message: %w", err)
	}
	return p.client.Publish(ctx, SubmissionQueue, data)
}
