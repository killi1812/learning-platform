package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var (
	ErrClientClosed  = errors.New("rabbitmq client is closed")
	ErrPublishFailed = errors.New("failed to publish rabbitmq message")
)

// DeadLetterQueueName returns the DLQ name for a given queue.
func DeadLetterQueueName(queueName string) string {
	return queueName + ".dead"
}

// QueueArgsWithDLQ returns the AMQP Table configured with dead letter routing to the queue's DLQ.
func QueueArgsWithDLQ(queueName string) amqp.Table {
	return amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": DeadLetterQueueName(queueName),
	}
}

// Client wraps an AMQP connection, publisher channel with confirms, and consumer utilities.
type Client struct {
	url       string
	conn      *amqp.Connection
	pubCh     *amqp.Channel
	pubMu     sync.Mutex
	isClosed  bool
	closeMu   sync.RWMutex
	logger    *zap.SugaredLogger
	reconnect chan struct{}
}

// NewClient dials RabbitMQ with bounded backoff retries and initializes the publisher in confirm mode.
func NewClient(url string) (*Client, error) {
	logger := zap.S().Named("rabbitmq")

	var conn *amqp.Connection
	var err error

	maxAttempts := 5
	backoff := 500 * time.Millisecond

	for i := 1; i <= maxAttempts; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		logger.Warnf("RabbitMQ dial attempt %d/%d failed: %v. Retrying in %v...", i, maxAttempts, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq at %s after %d attempts: %w", url, maxAttempts, err)
	}

	pubCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open publisher channel: %w", err)
	}

	if err := pubCh.Confirm(false); err != nil {
		_ = pubCh.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to put publisher channel into confirm mode: %w", err)
	}

	client := &Client{
		url:       url,
		conn:      conn,
		pubCh:     pubCh,
		logger:    logger,
		reconnect: make(chan struct{}, 1),
	}

	go client.monitorConnection()

	logger.Infof("Connected to RabbitMQ")
	return client, nil
}

func (c *Client) monitorConnection() {
	closeErr := make(chan *amqp.Error, 1)
	c.conn.NotifyClose(closeErr)

	err, ok := <-closeErr
	if !ok {
		return // Clean shutdown
	}

	c.closeMu.RLock()
	closed := c.isClosed
	c.closeMu.RUnlock()

	if closed {
		return
	}

	c.logger.Errorf("RabbitMQ connection lost: %v. Attempting to reconnect...", err)

	for {
		c.closeMu.RLock()
		if c.isClosed {
			c.closeMu.RUnlock()
			return
		}
		c.closeMu.RUnlock()

		conn, dialErr := amqp.Dial(c.url)
		if dialErr == nil {
			pubCh, chErr := conn.Channel()
			if chErr == nil && pubCh.Confirm(false) == nil {
				c.pubMu.Lock()
				_ = c.pubCh.Close()
				_ = c.conn.Close()
				c.conn = conn
				c.pubCh = pubCh
				c.pubMu.Unlock()

				c.logger.Infof("RabbitMQ successfully reconnected")
				select {
				case c.reconnect <- struct{}{}:
				default:
				}

				go c.monitorConnection()
				return
			}
			if pubCh != nil {
				_ = pubCh.Close()
			}
			_ = conn.Close()
		}

		time.Sleep(2 * time.Second)
	}
}

// DeclareQueue declares a durable queue along with its associated dead-letter queue.
func (c *Client) DeclareQueue(name string) error {
	c.pubMu.Lock()
	defer c.pubMu.Unlock()

	c.closeMu.RLock()
	if c.isClosed {
		c.closeMu.RUnlock()
		return ErrClientClosed
	}
	c.closeMu.RUnlock()

	// 1. Declare DLQ
	dlqName := DeadLetterQueueName(name)
	_, err := c.pubCh.QueueDeclare(
		dlqName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue %s: %w", dlqName, err)
	}

	// 2. Declare primary queue routing to DLQ
	args := QueueArgsWithDLQ(name)
	_, err = c.pubCh.QueueDeclare(
		name,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", name, err)
	}

	c.logger.Debugf("Declared queue %s with DLQ %s", name, dlqName)
	return nil
}

// Publish writes a persistent message with publisher confirmation.
func (c *Client) Publish(ctx context.Context, queue string, body []byte) error {
	c.pubMu.Lock()
	defer c.pubMu.Unlock()

	c.closeMu.RLock()
	if c.isClosed {
		c.closeMu.RUnlock()
		return ErrClientClosed
	}
	c.closeMu.RUnlock()

	deferred, err := c.pubCh.PublishWithDeferredConfirmWithContext(
		ctx,
		"",    // default exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	confirmed, err := deferred.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("failed waiting for publish confirmation: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("%w: broker rejected message (nack)", ErrPublishFailed)
	}

	return nil
}

// Consume runs a consumption loop with Qos prefetch on a dedicated channel, delivering messages to handler.
// The loop runs until ctx is canceled or the client is closed.
func (c *Client) Consume(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.closeMu.RLock()
		if c.isClosed {
			c.closeMu.RUnlock()
			return ErrClientClosed
		}
		conn := c.conn
		c.closeMu.RUnlock()

		ch, err := conn.Channel()
		if err != nil {
			c.logger.Errorf("Failed to open consumer channel for %s: %v. Retrying in 2s...", queue, err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}

		if err := ch.Qos(prefetch, 0, false); err != nil {
			_ = ch.Close()
			c.logger.Errorf("Failed to set QoS on consumer channel: %v. Retrying in 2s...", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}

		deliveries, err := ch.ConsumeWithContext(
			ctx,
			queue,
			"",    // consumer tag (auto-generated)
			false, // autoAck (manual ack required)
			false, // exclusive
			false, // noLocal
			false, // noWait
			nil,   // args
		)
		if err != nil {
			_ = ch.Close()
			c.logger.Errorf("Failed to start consumer on queue %s: %v. Retrying in 2s...", queue, err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}

		c.logger.Infof("Started consuming queue %s (prefetch: %d)", queue, prefetch)

		for d := range deliveries {
			handler(d)
		}

		_ = ch.Close()

		// Channel or connection closed, wait a moment or wait for reconnect signal
		select {
		case <-ctx.Done():
			return nil
		case <-c.reconnect:
			c.logger.Infof("Consumer detected reconnect, recreating consumer channel for %s...", queue)
		case <-time.After(1 * time.Second):
		}
	}
}

// Close gracefully shuts down the publisher channel and connection.
func (c *Client) Close() error {
	c.closeMu.Lock()
	if c.isClosed {
		c.closeMu.Unlock()
		return nil
	}
	c.isClosed = true
	c.closeMu.Unlock()

	c.pubMu.Lock()
	defer c.pubMu.Unlock()

	var errs []error
	if c.pubCh != nil {
		if err := c.pubCh.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing rabbitmq client: %v", errs)
	}
	return nil
}
