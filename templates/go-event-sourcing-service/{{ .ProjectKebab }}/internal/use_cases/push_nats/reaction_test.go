//go:build integration

package push_nats_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/natspush"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/push_nats"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

func TestReaction(t *testing.T) {
	t.Run("publish", func(t *testing.T) {
		t.Run("publishes the event to the push stream", func(t *testing.T) {
			url := os.Getenv("NATS_URL")
			require.NotEmpty(t, url, "NATS_URL must be set, see .env.integration")

			conn, err := nats.Connect(url)
			require.NoError(t, err)
			t.Cleanup(conn.Close)

			r, err := push_nats.New(conn)
			require.NoError(t, err)

			js, err := conn.JetStream()
			require.NoError(t, err)
			sub, err := js.SubscribeSync(natspush.Subject, nats.DeliverNew())
			require.NoError(t, err)
			t.Cleanup(func() { _ = sub.Unsubscribe() })

			evt := fake.NewSomethingHappenedEvent("stream-1", "value")
			require.NoError(t, r.Handle(context.Background(), &evt))

			msg, err := sub.NextMsg(5 * time.Second)
			require.NoError(t, err)

			var pushed natspush.Message
			require.NoError(t, json.Unmarshal(msg.Data, &pushed))
			require.Equal(t, evt.Type(), pushed.Type)
			require.Equal(t, map[string]any{"ID": "stream-1", "Value": "value"}, pushed.Data)
		})
	})
}
