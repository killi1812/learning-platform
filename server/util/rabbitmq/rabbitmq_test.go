package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConfirmation struct {
	confirmed bool
	err       error
}

func (m *mockConfirmation) WaitContext(ctx context.Context) (bool, error) {
	return m.confirmed, m.err
}

type mockChannel struct {
	mu                sync.Mutex
	confirmErr        error
	declareQueueErr   map[string]error
	declaredQueues    []string
	publishConfirm    amqpConfirmation
	publishErr        error
	publishedMessages [][]byte
	qosErr            error
	consumeCh         <-chan amqp.Delivery
	consumeErr        error
	closeErr          error
	closed            bool
}

func (m *mockChannel) Confirm(noWait bool) error {
	return m.confirmErr
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.declareQueueErr != nil {
		if err, ok := m.declareQueueErr[name]; ok && err != nil {
			return amqp.Queue{}, err
		}
	}
	m.declaredQueues = append(m.declaredQueues, name)
	return amqp.Queue{Name: name}, nil
}

func (m *mockChannel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (amqpConfirmation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishErr != nil {
		return nil, m.publishErr
	}
	m.publishedMessages = append(m.publishedMessages, msg.Body)
	if m.publishConfirm != nil {
		return m.publishConfirm, nil
	}
	return &mockConfirmation{confirmed: true}, nil
}

func (m *mockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return m.qosErr
}

func (m *mockChannel) ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	if m.consumeCh != nil {
		return m.consumeCh, nil
	}
	ch := make(chan amqp.Delivery)
	close(ch)
	return ch, nil
}

func (m *mockChannel) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
}

type mockConnection struct {
	mu          sync.Mutex
	channels    []amqpChannel
	channelErr  error
	closeErr    error
	closed      bool
	notifyChans []chan *amqp.Error
}

func (m *mockConnection) Channel() (amqpChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelErr != nil {
		return nil, m.channelErr
	}
	if len(m.channels) > 0 {
		ch := m.channels[0]
		if len(m.channels) > 1 {
			m.channels = m.channels[1:]
		}
		return ch, nil
	}
	return &mockChannel{}, nil
}

func (m *mockConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifyChans = append(m.notifyChans, receiver)
	return receiver
}

func (m *mockConnection) triggerClose(err *amqp.Error) {
	m.mu.Lock()
	chans := m.notifyChans
	m.notifyChans = nil
	m.mu.Unlock()
	for _, ch := range chans {
		if err != nil {
			ch <- err
		}
		close(ch)
	}
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	m.closed = true
	chans := m.notifyChans
	m.notifyChans = nil
	err := m.closeErr
	m.mu.Unlock()
	for _, ch := range chans {
		close(ch)
	}
	return err
}

func withDialer(d func(string) (amqpConnection, error)) ClientOption {
	return func(cfg *clientConfig) {
		cfg.dial = d
	}
}

func TestDeadLetterQueueName(t *testing.T) {
	assert.Equal(t, "submission.created.dead", DeadLetterQueueName("submission.created"))
	assert.Equal(t, "events.dead", DeadLetterQueueName("events"))
}

func TestQueueArgsWithDLQ(t *testing.T) {
	args := QueueArgsWithDLQ("submission.created")
	assert.Equal(t, "", args["x-dead-letter-exchange"])
	assert.Equal(t, "submission.created.dead", args["x-dead-letter-routing-key"])
}

func TestNewClient_InvalidUrlFailsGracefully(t *testing.T) {
	client, err := NewClient(
		"amqp://127.0.0.1:1/",
		WithRetry(2, time.Millisecond),
		WithRetryDelay(time.Millisecond),
	)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_LifecycleAndErrors(t *testing.T) {
	t.Run("Channel failure closes connection and returns error", func(t *testing.T) {
		conn := &mockConnection{channelErr: errors.New("channel open error")}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open publisher channel")
		assert.Nil(t, client)
		assert.True(t, conn.closed)
	})

	t.Run("Confirm failure closes channel and connection", func(t *testing.T) {
		ch := &mockChannel{confirmErr: errors.New("confirm mode unsupported")}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to put publisher channel into confirm mode")
		assert.Nil(t, client)
		assert.True(t, ch.closed)
		assert.True(t, conn.closed)
	})

	t.Run("Retries dial until success", func(t *testing.T) {
		attempts := 0
		conn := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		client, err := NewClient(
			"amqp://mock",
			WithRetry(3, time.Millisecond),
			withDialer(func(s string) (amqpConnection, error) {
				attempts++
				if attempts < 2 {
					return nil, errors.New("temporary dial error")
				}
				return conn, nil
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, 2, attempts)
		assert.NoError(t, client.Close())
	})
}

func TestClient_DeclareQueue(t *testing.T) {
	t.Run("Declares DLQ and primary queue", func(t *testing.T) {
		ch := &mockChannel{}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.DeclareQueue("submission.created")
		assert.NoError(t, err)
		assert.Equal(t, []string{"submission.created.dead", "submission.created"}, ch.declaredQueues)
	})

	t.Run("Fails when DLQ declaration fails", func(t *testing.T) {
		ch := &mockChannel{
			declareQueueErr: map[string]error{
				"submission.created.dead": errors.New("dlq error"),
			},
		}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.DeclareQueue("submission.created")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to declare dead letter queue")
	})

	t.Run("Fails when primary queue declaration fails", func(t *testing.T) {
		ch := &mockChannel{
			declareQueueErr: map[string]error{
				"submission.created": errors.New("primary queue error"),
			},
		}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.DeclareQueue("submission.created")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to declare queue")
	})

	t.Run("Returns ErrClientClosed when client is closed", func(t *testing.T) {
		conn := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		assert.NoError(t, client.Close())

		err = client.DeclareQueue("submission.created")
		assert.ErrorIs(t, err, ErrClientClosed)
	})
}

func TestClient_Publish(t *testing.T) {
	t.Run("Publishes and confirms message", func(t *testing.T) {
		ch := &mockChannel{}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.Publish(context.Background(), "submission.created", []byte(`{"id":1}`))
		assert.NoError(t, err)
		require.Len(t, ch.publishedMessages, 1)
		assert.Equal(t, []byte(`{"id":1}`), ch.publishedMessages[0])
	})

	t.Run("Returns error when publish fails", func(t *testing.T) {
		ch := &mockChannel{publishErr: errors.New("socket write failed")}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.Publish(context.Background(), "submission.created", []byte(`{}`))
		assert.ErrorIs(t, err, ErrPublishFailed)
	})

	t.Run("Returns error when confirmation wait fails", func(t *testing.T) {
		ch := &mockChannel{
			publishConfirm: &mockConfirmation{err: context.DeadlineExceeded},
		}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.Publish(context.Background(), "submission.created", []byte(`{}`))
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Returns error when broker nacks message", func(t *testing.T) {
		ch := &mockChannel{
			publishConfirm: &mockConfirmation{confirmed: false},
		}
		conn := &mockConnection{channels: []amqpChannel{ch}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		err = client.Publish(context.Background(), "submission.created", []byte(`{}`))
		assert.ErrorIs(t, err, ErrPublishFailed)
		assert.Contains(t, err.Error(), "nack")
	})

	t.Run("Returns ErrClientClosed when client is closed", func(t *testing.T) {
		conn := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		assert.NoError(t, client.Close())

		err = client.Publish(context.Background(), "submission.created", []byte(`{}`))
		assert.ErrorIs(t, err, ErrClientClosed)
	})
}

func TestClient_ConsumeAndReconnect(t *testing.T) {
	t.Run("Consumes deliveries and exits when ctx is canceled", func(t *testing.T) {
		deliveries := make(chan amqp.Delivery, 2)
		deliveries <- amqp.Delivery{Body: []byte("msg1")}
		deliveries <- amqp.Delivery{Body: []byte("msg2")}
		close(deliveries)

		pubCh := &mockChannel{}
		consCh := &mockChannel{consumeCh: deliveries}
		conn := &mockConnection{channels: []amqpChannel{pubCh, consCh}}

		client, err := NewClient(
			"amqp://mock",
			WithRetryDelay(time.Millisecond),
			withDialer(func(s string) (amqpConnection, error) {
				return conn, nil
			}),
		)
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		ctx, cancel := context.WithCancel(context.Background())
		var received []string
		err = client.Consume(ctx, "submission.created", 10, func(d amqp.Delivery) {
			received = append(received, string(d.Body))
			if len(received) == 2 {
				cancel()
			}
		})
		assert.NoError(t, err)
		assert.Equal(t, []string{"msg1", "msg2"}, received)
	})

	t.Run("Returns ErrClientClosed when client is closed before consuming", func(t *testing.T) {
		conn := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
			return conn, nil
		}))
		require.NoError(t, err)
		assert.NoError(t, client.Close())

		err = client.Consume(context.Background(), "submission.created", 10, func(d amqp.Delivery) {})
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("Retries on channel, QoS, and ConsumeWithContext errors then exits on ctx cancel", func(t *testing.T) {
		pubCh := &mockChannel{}
		qosErrCh := &mockChannel{qosErr: errors.New("qos fail")}
		consumeErrCh := &mockChannel{consumeErr: errors.New("consume fail")}
		conn := &mockConnection{channels: []amqpChannel{pubCh, qosErrCh, consumeErrCh}}

		client, err := NewClient(
			"amqp://mock",
			WithRetryDelay(time.Millisecond),
			withDialer(func(s string) (amqpConnection, error) {
				return conn, nil
			}),
		)
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err = client.Consume(ctx, "submission.created", 10, func(d amqp.Delivery) {})
		assert.NoError(t, err)

		// Also test conn.Channel() error path in Consume
		conn.mu.Lock()
		conn.channelErr = errors.New("conn channel error")
		conn.mu.Unlock()

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel2()
		err = client.Consume(ctx2, "submission.created", 10, func(d amqp.Delivery) {})
		assert.NoError(t, err)
	})

	t.Run("MonitorConnection reconnects on unexpected connection loss", func(t *testing.T) {
		conn1 := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		conn2 := &mockConnection{channels: []amqpChannel{&mockChannel{}}}
		badConn := &mockConnection{channels: []amqpChannel{&mockChannel{confirmErr: errors.New("confirm fail")}}}

		var mu sync.Mutex
		dialCount := 0

		client, err := NewClient(
			"amqp://mock",
			WithRetryDelay(time.Millisecond),
			withDialer(func(s string) (amqpConnection, error) {
				mu.Lock()
				defer mu.Unlock()
				dialCount++
				switch dialCount {
				case 1:
					return conn1, nil
				case 2:
					return nil, errors.New("dial fail")
				case 3:
					return badConn, nil
				default:
					return conn2, nil
				}
			}),
		)
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		time.Sleep(2 * time.Millisecond)
		conn1.triggerClose(&amqp.Error{Code: 504, Reason: "channel/connection is not open"})

		select {
		case <-client.reconnect:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("expected client to reconnect")
		}

		mu.Lock()
		assert.GreaterOrEqual(t, dialCount, 4)
		mu.Unlock()
	})
}

func TestClient_CloseErrors(t *testing.T) {
	ch := &mockChannel{closeErr: errors.New("channel close err")}
	conn := &mockConnection{
		channels: []amqpChannel{ch},
		closeErr: errors.New("conn close err"),
	}
	client, err := NewClient("amqp://mock", withDialer(func(s string) (amqpConnection, error) {
		return conn, nil
	}))
	require.NoError(t, err)

	err = client.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "errors closing rabbitmq client")

	// Second close is a no-op
	assert.NoError(t, client.Close())
}
