package amqp_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/rabbitmq/amqp091-go"
	"github.com/reddec/web-form/internal/notifications/amqp"
	"github.com/reddec/web-form/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var amqpURL string

func TestAMQP_Run(t *testing.T) {
	factory := amqp.New(amqpURL, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go factory.Run(ctx)

	t.Run("simple", func(t *testing.T) {
		require.NoError(t, define("", t.Name(), ""))

		notify := factory.Create(schema.AMQP{
			Key: mustTemplate(t.Name()), // when exchange not specified, routing key is queue name
		})

		err := notify.Dispatch(ctx, &Event{
			result: map[string]any{"Name": t.Name()},
		})
		require.NoError(t, err)

		msg, err := getMessage(ctx, t.Name())
		require.NoError(t, err)

		assert.Equal(t, `{"Name":"`+t.Name()+`"}`, string(msg.Body))
	})

	t.Run("full", func(t *testing.T) {
		require.NoError(t, define("test", t.Name(), "full"))

		notify := factory.Create(schema.AMQP{
			Exchange: "test",
			Key:      mustTemplate("full"),
			Headers: map[string]string{
				"X-Hello": "World",
			},
			Correlation: mustTemplate("reply-{{.Result.ID}}"),
			ID:          mustTemplate("{{.Result.ID}}"),
			Message:     mustTemplate("{{.Result.Name}}"),
		})

		err := notify.Dispatch(ctx, &Event{
			result: map[string]any{"Name": t.Name(), "ID": 1234},
		})
		require.NoError(t, err)

		msg, err := getMessage(ctx, t.Name())
		require.NoError(t, err)

		assert.Equal(t, t.Name(), string(msg.Body))
		assert.Equal(t, "1234", msg.MessageId)
		assert.Equal(t, "reply-1234", msg.CorrelationId)
		assert.Equal(t, amqp091.Table{
			"X-Hello": "World",
		}, msg.Headers)
		assert.Equal(t, "test", msg.Exchange)
		assert.Equal(t, "full", msg.RoutingKey)
	})
}

func getMessage(ctx context.Context, queue string) (amqp091.Delivery, error) {
	con, err := amqp091.DialConfig(amqpURL, amqp091.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		},
	})
	if err != nil {
		return amqp091.Delivery{}, fmt.Errorf("dial: %w", err)
	}
	defer con.Close()

	ch, err := con.Channel()
	if err != nil {
		return amqp091.Delivery{}, fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	m, err := ch.ConsumeWithContext(ctx, queue, "", true, false, false, false, nil)
	if err != nil {
		return amqp091.Delivery{}, fmt.Errorf("consume: %w", err)
	}

	select {
	case v, ok := <-m:
		if !ok {
			return amqp091.Delivery{}, fmt.Errorf("consumer closed")
		}
		return v, nil
	case <-ctx.Done():
		return amqp091.Delivery{}, ctx.Err()
	}
}

func define(exchange string, queue string, routingKey string) error {
	con, err := amqp091.DialConfig(amqpURL, amqp091.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		},
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer con.Close()

	ch, err := con.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	if exchange != "" {
		if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
			return fmt.Errorf("decalre exchange: %w", err)
		}
	}

	if queue != "" {
		if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return fmt.Errorf("decalre queue: %w", err)
		}
	}

	if routingKey != "" && queue != "" && exchange != "" {
		if err := ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue: %w", err)
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("rabbitmq", "3.12", []string{})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		amqpURL = fmt.Sprintf("amqp://guest:guest@localhost:%s", resource.GetPort("5672/tcp"))

		con, err := amqp091.DialConfig(amqpURL, amqp091.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, 5*time.Second)
			},
		})
		if err != nil {
			return err
		}
		_ = con.Close()
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func mustTemplate(text string) *schema.Template {
	var t schema.Template
	err := t.UnmarshalText([]byte(text))
	if err != nil {
		panic(err)
	}
	return &t
}

type Event struct {
	result     map[string]any
	error      error
	definition schema.Form
}

func (ev *Event) Form() *schema.Form {
	return &ev.definition
}

func (ev *Event) Error() error {
	return ev.error
}

func (ev *Event) Result() map[string]any {
	return ev.result
}
