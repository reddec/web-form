package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/reddec/web-form/internal/notifications"
	"github.com/reddec/web-form/internal/schema"
)

var (
	ErrNonSuccessCode = errors.New("non-2xx response code")
)

const (
	defaultTimeout  = 10 * time.Second
	defaultRetries  = 3
	defaultInterval = 15 * time.Second
	defaultMethod   = http.MethodPost
)

func New(buffer int) *Dispatcher {
	return &Dispatcher{tasks: make(chan webhookTask, buffer)}
}

type Dispatcher struct {
	tasks chan webhookTask
}

func (wd *Dispatcher) Create(webhook schema.Webhook) notifications.Notification {
	if webhook.Timeout <= 0 {
		webhook.Timeout = defaultTimeout
	}
	if webhook.Retry == 0 {
		webhook.Retry = defaultRetries
	}
	if webhook.Interval <= 0 {
		webhook.Interval = defaultInterval
	}
	if webhook.Method == "" {
		webhook.Method = defaultMethod
	}

	return notifications.NotificationFunc(func(ctx context.Context, event notifications.Event) error {
		payload, err := notifications.RenderPayload(webhook.Message, event)
		if err != nil {
			return fmt.Errorf("render webhook: %w", err)
		}
		return wd.enqueue(ctx, webhook, payload)
	})
}

func (wd *Dispatcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		select {
		case task, ok := <-wd.tasks:
			if !ok {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				task.Send(ctx)
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (wd *Dispatcher) enqueue(ctx context.Context, webhook schema.Webhook, payload []byte) error {
	select {
	case wd.tasks <- webhookTask{webhook: webhook, payload: payload}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

type webhookTask struct {
	webhook schema.Webhook
	payload []byte
}

func (wt *webhookTask) Send(global context.Context) {
	u, err := url.Parse(wt.webhook.URL)
	if err != nil {
		slog.Error("webhook url invalid - skipping", "url", wt.webhook.URL, "error", err)
		return
	}
	var attempt int
	for {
		if err := wt.trySend(global); err != nil {
			slog.Warn("failed deliver webhook", "url", u.Redacted(), "error", err, "attempt", attempt+1, "retries", wt.webhook.Retry, "retry-after", wt.webhook.Interval)
		} else {
			slog.Info("webhook delivered", "url", u.Redacted(), "attempt", attempt+1, "retries", wt.webhook.Retry)
			break
		}

		if attempt >= wt.webhook.Retry {
			break
		}
		select {
		case <-time.After(wt.webhook.Timeout):
		case <-global.Done():
			slog.Info("webhook retry stopped due to global context stop")
			return
		}
		attempt++
	}
}

func (wt *webhookTask) trySend(global context.Context) error {
	ctx, cancel := context.WithTimeout(global, wt.webhook.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, wt.webhook.Method, wt.webhook.URL, bytes.NewReader(wt.payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for k, v := range wt.webhook.Headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	_, _ = io.Copy(io.Discard, res.Body) // drain content to keep connection healthy

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("%w: %d", ErrNonSuccessCode, res.StatusCode)
	}
	return nil
}
