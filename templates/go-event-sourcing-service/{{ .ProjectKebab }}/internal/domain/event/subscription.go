package event

import (
	"context"
	"errors"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

var ErrSubscriptionEnded = errors.New("event subscription ended")

type Subscription interface {
	SubscribeToAll(ctx context.Context, after *domain.Position, eventTypes []string) (<-chan domain.Event, error)
	LastPosition(ctx context.Context) (*domain.Position, error)
}
