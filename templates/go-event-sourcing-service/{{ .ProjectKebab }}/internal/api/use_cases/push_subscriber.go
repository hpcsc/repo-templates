package use_cases

import (
	"context"
	"log/slog"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/websocket"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/jackc/pgx/v5/pgxpool"
)

func startPushSubscriber(ctx context.Context, pool *pgxpool.Pool, reg *registry.OfEvents, hub *websocket.Hub, checkpoints checkpoint.Store, logger *slog.Logger) {
	sub := subscription.NewPostgres(pool, reg, logger, subscription.WithListenChannel(domainEvent.PushChannel))

	after, err := checkpoints.Get(ctx, "push-subscriber")
	if err != nil {
		logger.Error("failed to get push subscriber checkpoint", "error", err)
		return
	}
	if after == nil {
		after, err = sub.LastPosition(ctx)
		if err != nil {
			logger.Error("failed to get last position for push subscription", "error", err)
			return
		}
	}

	events, err := sub.SubscribeToAll(ctx, after, nil)
	if err != nil {
		logger.Error("failed to subscribe to push events", "error", err)
		return
	}

	for evt := range events {
		hub.Broadcast(context.Background(), websocket.Message{
			Type: evt.Type(),
			Data: evt.Payload,
		})

		if err := checkpoints.Set(ctx, "push-subscriber", evt.Position); err != nil {
			logger.Warn("failed to save push subscriber checkpoint", "error", err)
		}
	}

	logger.Info("push event subscription stopped")
}