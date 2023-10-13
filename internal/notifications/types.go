package notifications

import (
	"context"

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
	Form() schema.Form
	Error() error
	Result() map[string]any
}
