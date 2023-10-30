package amqp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/reddec/web-form/internal/notifications"
	"github.com/reddec/web-form/internal/schema"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultRetries  = 3
	defaultInterval = 15 * time.Second
)

// copied from amp091 defaults.
const (
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"
)

func New(url string, buffer int) *AMQP {
	return &AMQP{url: url, tasks: make(chan task, buffer)}
}

type AMQP struct {
	url   string
	tasks chan task
}

//nolint:cyclop
func (amqp *AMQP) Create(definition schema.AMQP) notifications.Notification {
	if definition.Timeout <= 0 {
		definition.Timeout = defaultTimeout
	}
	if definition.Retry == 0 {
		definition.Retry = defaultRetries
	}
	if definition.Interval <= 0 {
		definition.Interval = defaultInterval
	}
	if !definition.Message.Valid {
		// nil message causes JSON payload
		if definition.Type == "" {
			definition.Type = "application/json"
		}
		definition.Message = schema.MustTemplate[schema.NotifyContext]("{{.Result | toJson}}")
	}

	return notifications.NotificationFunc(func(ctx context.Context, event schema.NotifyContext) error {
		payload, err := definition.Message.Bytes(&event)
		if err != nil {
			return fmt.Errorf("render payload: %w", err)
		}

		key, err := definition.Key.String(&event)
		if err != nil {
			return fmt.Errorf("render routing key: %w", err)
		}

		correlationID, err := definition.Correlation.String(&event)
		if err != nil {
			return fmt.Errorf("render correlation ID: %w", err)
		}

		messageID, err := definition.ID.String(&event)
		if err != nil {
			return fmt.Errorf("render message ID: %w", err)
		}

		t := task{
			definition:    definition,
			key:           key,
			correlationID: correlationID,
			messageID:     messageID,
			payload:       payload,
		}

		select {
		case amqp.tasks <- t:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (amqp *AMQP) Run(ctx context.Context) {
	w := &worker{url: amqp.url}
	defer w.close()

	for {
		select {
		case t, ok := <-amqp.tasks:
			if !ok {
				return
			}
			amqp.sendTask(ctx, w, t)
		case <-ctx.Done():
			return
		}
	}
}

func (amqp *AMQP) sendTask(ctx context.Context, w *worker, t task) {
	var attempt int
	logger := slog.With("routing-key", t.key, "exchange", t.definition.Exchange, "message-id", t.messageID)
	for {
		if err := amqp.trySendTask(ctx, w, t); err != nil {
			w.close() // reset state on error
			logger.Warn("failed publish AMQP message", "error", err, "attempt", attempt+1, "retries", t.definition.Retry, "retry-after", t.definition.Interval)
		} else {
			logger.Info("message published to AMQP broker", "attempt", attempt+1, "retries", t.definition.Retry)
			break
		}

		if attempt >= t.definition.Retry {
			break
		}
		select {
		case <-time.After(t.definition.Timeout):
		case <-ctx.Done():
			logger.Info("publishing stopped due to global context stop")
			return
		}
		attempt++
	}
}

func (amqp *AMQP) trySendTask(global context.Context, w *worker, t task) error {
	ctx, cancel := context.WithTimeout(global, t.definition.Timeout)
	defer cancel()

	ch, err := w.getChannel(ctx)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}

	headers := make(amqp091.Table, len(t.definition.Headers))
	for k, v := range t.definition.Headers {
		headers[k] = v
	}

	return ch.PublishWithContext(ctx, t.definition.Exchange, t.key, false, false, amqp091.Publishing{
		MessageId:     t.messageID,
		CorrelationId: t.correlationID,
		Timestamp:     time.Now(),
		Headers:       headers,
		ContentType:   t.definition.Type,
		Body:          t.payload,
	})
}

type worker struct {
	url        string
	connection *amqp091.Connection
	channel    *amqp091.Channel
}

func (worker *worker) getChannel(ctx context.Context) (*amqp091.Channel, error) {
	if worker.channel != nil {
		return worker.channel, nil
	}

	connection, err := worker.getConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		worker.close()
		return nil, fmt.Errorf("allocate channel: %w", err)
	}
	worker.channel = channel

	return channel, nil
}

func (worker *worker) getConnection(ctx context.Context) (*amqp091.Connection, error) {
	if worker.connection != nil {
		return worker.connection, nil
	}

	conn, err := amqp091.DialConfig(worker.url, amqp091.Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
		Dial: func(network, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dial broker: %w", err)
	}
	worker.connection = conn
	return conn, nil
}

func (worker *worker) close() {
	if worker.channel != nil {
		_ = worker.channel.Close()
	}
	worker.channel = nil

	if worker.connection != nil {
		_ = worker.connection.Close()
	}
	worker.connection = nil
}

type task struct {
	definition    schema.AMQP
	key           string
	correlationID string
	messageID     string
	payload       []byte
}
