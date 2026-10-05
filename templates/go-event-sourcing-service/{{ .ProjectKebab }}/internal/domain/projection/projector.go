package projection

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/consumer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

const DefaultBatchSize = 100

var _ consumer.Handler = (*projector)(nil)

type ProjectorOption func(*projector)

func WithBatchSize(batchSize int) ProjectorOption {
	return func(p *projector) {
		p.batchSize = batchSize
	}
}

func NewProjector(
	projection Interface,
	db DB,
	checkpointStore checkpoint.Store,
	subscription event.Subscription,
	logger *slog.Logger,
	opts ...ProjectorOption,
) *consumer.Consumer {
	p := &projector{
		projection:      projection,
		db:              db,
		checkpointStore: checkpointStore,
		batchSize:       DefaultBatchSize,
	}

	for _, opt := range opts {
		opt(p)
	}

	return consumer.New(p, p.batchSize, checkpointStore, subscription, logger)
}

type projector struct {
	projection      Interface
	db              DB
	checkpointStore checkpoint.Store
	batchSize       int
}

func (p *projector) Name() string {
	return p.projection.Name()
}

func (p *projector) EventTypes() []string {
	return p.projection.EventTypes()
}

func (p *projector) StartWithoutCheckpoint(context.Context, event.Subscription) (*domain.Position, error) {
	return nil, nil
}

func (p *projector) HandleBatch(ctx context.Context, events []domain.Event) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for projection %s: %w", p.Name(), err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i := range events {
		evt := &events[i]
		if err := p.projection.Handle(ctx, tx, evt); err != nil {
			return fmt.Errorf("projection %s failed to handle event %s at version %d of stream %s: %w", p.Name(), evt.Type(), evt.Version, evt.StreamID, err)
		}
	}

	if err := p.checkpointStore.SetInTx(ctx, tx, p.Name(), events[len(events)-1].Position); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction for projection %s: %w", p.Name(), err)
	}
	return nil
}
