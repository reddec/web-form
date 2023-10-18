package notifications

import (
	"context"
	"encoding/json"

	"github.com/reddec/web-form/internal/schema"
)

type Notification interface {
	Dispatch(ctx context.Context, event Event) error
}

type NotificationFunc func(ctx context.Context, event Event) error

func (nf NotificationFunc) Dispatch(ctx context.Context, event Event) error {
	return nf(ctx, event)
}

type Event interface {
	Form() *schema.Form
	Error() error
	Result() map[string]any
}

func RenderPayload(message *schema.Template, event Event) ([]byte, error) {
	if message == nil {
		return json.Marshal(event.Result())
	}
	return message.Render(event)
}
