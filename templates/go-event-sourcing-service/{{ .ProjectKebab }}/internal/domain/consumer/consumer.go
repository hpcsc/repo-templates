package consumer

import (
	"context"
	"log/slog"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type Handler interface {
	Name() string
	EventTypes() []string
	StartWithoutCheckpoint(ctx context.Context, subscription event.Subscription) (*domain.Position, error)
	HandleBatch(ctx context.Context, events []domain.Event) error
}

type Consumer struct {
	handler         Handler
	batchSize       int
	checkpointStore checkpoint.Store
	subscription    event.Subscription
	logger          *slog.Logger
}

func New(handler Handler, batchSize int, checkpointStore checkpoint.Store, subscription event.Subscription, logger *slog.Logger) *Consumer {
	return &Consumer{
		handler:         handler,
		batchSize:       batchSize,
		checkpointStore: checkpointStore,
		subscription:    subscription,
		logger:          logger.With("consumer", handler.Name()),
	}
}

func (c *Consumer) Name() string {
	return c.handler.Name()
}

func (c *Consumer) Start(ctx context.Context) error {
	after, err := c.startPosition(ctx)
	if err != nil {
		return err
	}

	c.logger.Info("starting", "afterPosition", after)

	events, err := c.subscription.SubscribeToAll(ctx, after, c.handler.EventTypes())
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-events:
			if !ok {
				return c.subscriptionEnded(ctx)
			}

			batch, open := c.collectBatch(evt, events)
			if err := c.handler.HandleBatch(ctx, batch); err != nil {
				c.logger.Error("failed to handle events", "error", err, "firstPosition", batch[0].Position)
				return err
			}

			if !open {
				return c.subscriptionEnded(ctx)
			}
		}
	}
}

func (c *Consumer) startPosition(ctx context.Context) (*domain.Position, error) {
	after, err := c.checkpointStore.Get(ctx, c.handler.Name())
	if err != nil || after != nil {
		return after, err
	}

	start, err := c.handler.StartWithoutCheckpoint(ctx, c.subscription)
	if err != nil || start == nil {
		return start, err
	}

	// save it now: after a crash before the first event, a later end of the store skips events
	if err := c.checkpointStore.Set(ctx, c.handler.Name(), *start); err != nil {
		return nil, err
	}
	return start, nil
}

func (c *Consumer) collectBatch(first domain.Event, events <-chan domain.Event) ([]domain.Event, bool) {
	batch := []domain.Event{first}
	for len(batch) < c.batchSize {
		select {
		case evt, ok := <-events:
			if !ok {
				return batch, false
			}
			batch = append(batch, evt)
		default:
			return batch, true
		}
	}
	return batch, true
}

func (c *Consumer) subscriptionEnded(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	return event.ErrSubscriptionEnded
}
