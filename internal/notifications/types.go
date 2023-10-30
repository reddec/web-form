package notifications

import (
	"context"

	"github.com/reddec/web-form/internal/schema"
)

type Notification interface {
	Dispatch(ctx context.Context, event schema.NotifyContext) error
}

type NotificationFunc func(ctx context.Context, event schema.NotifyContext) error

func (nf NotificationFunc) Dispatch(ctx context.Context, event schema.NotifyContext) error {
	return nf(ctx, event)
}
