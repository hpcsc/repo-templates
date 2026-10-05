//go:build unit

package dispatch_test

import (
	"testing"

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
