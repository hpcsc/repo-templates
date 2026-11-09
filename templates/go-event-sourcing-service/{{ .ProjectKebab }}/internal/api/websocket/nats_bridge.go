package websocket

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/natspush"
	"github.com/nats-io/nats.go"
)

func BridgeNATS(ctx context.Context, conn *nats.Conn, hub *Hub, logger *slog.Logger) error {
	js, err := conn.JetStream()
	if err != nil {
		return err
	}
	if err := natspush.EnsureStream(js); err != nil {
		return err
	}

	sub, err := js.Subscribe(natspush.Subject, func(msg *nats.Msg) {
		var pushed natspush.Message
		if err := json.Unmarshal(msg.Data, &pushed); err != nil {
			logger.Warn("failed to unmarshal nats push message", "error", err)
			return
		}

		hub.Broadcast(ctx, Message{Type: pushed.Type, Data: pushed.Data})
	}, nats.DeliverNew())
	if err != nil {
		return err
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	logger.Info("nats push bridge started", "subject", natspush.Subject)

	<-ctx.Done()
	return nil
}
