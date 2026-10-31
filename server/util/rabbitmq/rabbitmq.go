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

// IClient defines operations for declaring queues, publishing, and consuming messages via RabbitMQ.
type IClient interface {
	DeclareQueue(name string) error
	Publish(ctx context.Context, queue string, body []byte) error
	Consume(ctx context.Context, queue string, prefetch int, handler func(d amqp.Delivery)) error
	Close() error
}

type amqpConfirmation interface {
	WaitContext(ctx context.Context) (bool, error)
}

type amqpChannel interface {
	Confirm(noWait bool) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (amqpConfirmation, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

type amqpConnection interface {
	Channel() (amqpChannel, error)
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Close() error
}

type realChannel struct {
	*amqp.Channel
}

func (ch *realChannel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (amqpConfirmation, error) {
	return ch.Channel.PublishWithDeferredConfirmWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

type realConnection struct {
	*amqp.Connection
}

func (c *realConnection) Channel() (amqpChannel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return &realChannel{Channel: ch}, nil
}

func defaultDial(url string) (amqpConnection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &realConnection{Connection: conn}, nil
}

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	maxAttempts    int
	initialBackoff time.Duration
	retryDelay     time.Duration
	dial           func(string) (amqpConnection, error)
}

// WithRetry configures the connection retry attempts and initial backoff duration.
func WithRetry(maxAttempts int, initialBackoff time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		if maxAttempts > 0 {
			cfg.maxAttempts = maxAttempts
		}
		if initialBackoff >= 0 {
			cfg.initialBackoff = initialBackoff
		}
	}
}

// WithRetryDelay configures the delay between reconnection and consumer channel retry attempts.
func WithRetryDelay(delay time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		if delay >= 0 {
			cfg.retryDelay = delay
		}
	}
}

// Client wraps an AMQP connection, publisher channel with confirms, and consumer utilities.
type Client struct {
	url        string
	conn       amqpConnection
	pubCh      amqpChannel
	pubMu      sync.Mutex
	isClosed   bool
	closeMu    sync.RWMutex
	logger     *zap.SugaredLogger
	reconnect  chan struct{}
	retryDelay time.Duration
	dial       func(string) (amqpConnection, error)
}

// NewClient dials RabbitMQ with bounded backoff retries and initializes the publisher in confirm mode.
func NewClient(url string, opts ...ClientOption) (*Client, error) {
	logger := zap.S().Named("rabbitmq")

	cfg := clientConfig{
		maxAttempts:    5,
		initialBackoff: 500 * time.Millisecond,
		retryDelay:     2 * time.Second,
		dial:           defaultDial,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var conn amqpConnection
	var err error

	backoff := cfg.initialBackoff

	for i := 1; i <= cfg.maxAttempts; i++ {
		conn, err = cfg.dial(url)
		if err == nil {
			break
		}
		logger.Warnf("RabbitMQ dial attempt %d/%d failed: %v. Retrying in %v...", i, cfg.maxAttempts, err, backoff)
		if i < cfg.maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq at %s after %d attempts: %w", url, cfg.maxAttempts, err)
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
		url:        url,
		conn:       conn,
		pubCh:      pubCh,
		logger:     logger,
		reconnect:  make(chan struct{}, 1),
		retryDelay: cfg.retryDelay,
		dial:       cfg.dial,
	}

	go client.monitorConnection()

	logger.Infof("Connected to RabbitMQ")
	return client, nil
}

func (c *Client) getConnection() (amqpConnection, bool) {
	c.closeMu.RLock()
	if c.isClosed {
		c.closeMu.RUnlock()
		return nil, true
	}
	c.closeMu.RUnlock()

	c.pubMu.Lock()
	conn := c.conn
	c.pubMu.Unlock()
	return conn, false
}

func (c *Client) monitorConnection() {
	conn, closed := c.getConnection()
	if closed || conn == nil {
		return
	}

	closeErr := make(chan *amqp.Error, 1)
	conn.NotifyClose(closeErr)

	err, ok := <-closeErr
	if !ok {
		return // Clean shutdown
	}

	c.closeMu.RLock()
	isClosed := c.isClosed
	c.closeMu.RUnlock()

	if isClosed {
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

		newConn, dialErr := c.dial(c.url)
		if dialErr == nil {
			pubCh, chErr := newConn.Channel()
			if chErr == nil && pubCh.Confirm(false) == nil {
				c.pubMu.Lock()
				_ = c.pubCh.Close()
				_ = c.conn.Close()
				c.conn = newConn
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
			_ = newConn.Close()
		}

		time.Sleep(c.retryDelay)
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
	c.closeMu.RLock()
	if c.isClosed {
		c.closeMu.RUnlock()
		return ErrClientClosed
	}
	c.closeMu.RUnlock()

	c.pubMu.Lock()
	ch := c.pubCh
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}

	deferred, err := ch.PublishWithDeferredConfirmWithContext(
		ctx,
		"",    // default exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		msg,
	)
	if err != nil && c.conn != nil {
		c.logger.Warnf("Publish failed on publisher channel (%v), attempting to reopen channel...", err)
		if newCh, openErr := c.conn.Channel(); openErr == nil && newCh.Confirm(false) == nil {
			_ = c.pubCh.Close()
			c.pubCh = newCh
			ch = newCh
			deferred, err = ch.PublishWithDeferredConfirmWithContext(
				ctx,
				"",
				queue,
				false,
				false,
				msg,
			)
		}
	}
	c.pubMu.Unlock()

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

		conn, closed := c.getConnection()
		if closed {
			return ErrClientClosed
		}

		ch, err := conn.Channel()
		if err != nil {
			c.logger.Errorf("Failed to open consumer channel for %s: %v. Retrying in %v...", queue, err, c.retryDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(c.retryDelay):
				continue
			}
		}

		if err := ch.Qos(prefetch, 0, false); err != nil {
			_ = ch.Close()
			c.logger.Errorf("Failed to set QoS on consumer channel: %v. Retrying in %v...", err, c.retryDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(c.retryDelay):
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
			c.logger.Errorf("Failed to start consumer on queue %s: %v. Retrying in %v...", queue, err, c.retryDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(c.retryDelay):
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
		case <-time.After(c.retryDelay):
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
