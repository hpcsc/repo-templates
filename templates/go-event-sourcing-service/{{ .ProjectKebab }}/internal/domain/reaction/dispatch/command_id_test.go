//go:build unit

package dispatch_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/dispatch"
	"github.com/stretchr/testify/require"
)

func TestCommandIDFor(t *testing.T) {
	t.Run("gives the same ID to the same event of the same reaction", func(t *testing.T) {
		first := dispatch.CommandIDFor("welcome-bonus", &domain.Event{ID: "event-1"})
		again := dispatch.CommandIDFor("welcome-bonus", &domain.Event{ID: "event-1"})

		require.Equal(t, first, again)
	})

	t.Run("gives another ID to another event", func(t *testing.T) {
		first := dispatch.CommandIDFor("welcome-bonus", &domain.Event{ID: "event-1"})
		other := dispatch.CommandIDFor("welcome-bonus", &domain.Event{ID: "event-2"})

		require.NotEqual(t, first, other)
	})

	t.Run("gives another ID when another reaction handles the same event", func(t *testing.T) {
		first := dispatch.CommandIDFor("welcome-bonus", &domain.Event{ID: "event-1"})
		other := dispatch.CommandIDFor("loyalty-points", &domain.Event{ID: "event-1"})

		require.NotEqual(t, first, other)
	})
}

func TestCommandFor(t *testing.T) {
	t.Run("gives the command the correlation ID of the event, and the event as its cause", func(t *testing.T) {
		id, correlationID := uuid.New(), uuid.New()
		evt := &domain.Event{ID: id.String(), Metadata: &domain.EventMetadata{CorrelationID: correlationID}}

		cmd := dispatch.CommandFor("welcome-bonus", evt, "payload")

		require.Equal(t, domain.Command[string]{
			ID:            dispatch.CommandIDFor("welcome-bonus", evt),
			CorrelationID: correlationID,
			CausationID:   id,
			Payload:       "payload",
		}, cmd)
	})
}
