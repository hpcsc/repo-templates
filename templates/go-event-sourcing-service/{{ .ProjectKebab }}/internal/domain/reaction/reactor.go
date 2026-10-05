package reaction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/consumer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

var _ consumer.Handler = (*reactor)(nil)

type ReactorOption func(*reactor)

func StartAtBeginning() ReactorOption {
	return func(r *reactor) {
		r.startAtBeginning = true
	}
}

func NewReactor(
	reaction Interface,
	checkpointStore checkpoint.Store,
	subscription event.Subscription,
	logger *slog.Logger,
	opts ...ReactorOption,
) *consumer.Consumer {
	r := &reactor{
		reaction:        reaction,
		checkpointStore: checkpointStore,
	}

	for _, opt := range opts {
		opt(r)
	}

	return consumer.New(r, 1, checkpointStore, subscription, logger)
}

type reactor struct {
	reaction         Interface
	checkpointStore  checkpoint.Store
	startAtBeginning bool
}

func (r *reactor) Name() string {
	return r.reaction.Name()
}

func (r *reactor) EventTypes() []string {
	return r.reaction.EventTypes()
}

func (r *reactor) StartWithoutCheckpoint(ctx context.Context, subscription event.Subscription) (*domain.Position, error) {
	if r.startAtBeginning {
		return nil, nil
	}
	return subscription.LastPosition(ctx)
}

func (r *reactor) HandleBatch(ctx context.Context, events []domain.Event) error {
	for i := range events {
		evt := &events[i]
		if err := r.reaction.Handle(ctx, evt); err != nil {
			return fmt.Errorf("reaction %s failed to handle event %s at version %d of stream %s: %w", r.Name(), evt.Type(), evt.Version, evt.StreamID, err)
		}

		if err := r.checkpointStore.Set(ctx, r.Name(), evt.Position); err != nil {
			return err
		}
	}
	return nil
}
