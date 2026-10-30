//go:build unit

package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	t.Run("cause", func(t *testing.T) {
		t.Run("a command with no correlation ID starts a chain with its own ID", func(t *testing.T) {
			cmd := domain.NewCommand("payload")

			require.Equal(t, domain.Cause{CommandID: cmd.ID, CorrelationID: cmd.ID, CausationID: cmd.ID}, cmd.Cause())
		})

		t.Run("keeps the correlation ID and the causation ID that the command carries", func(t *testing.T) {
			cmd := domain.Command[string]{ID: uuid.New(), CorrelationID: uuid.New(), CausationID: uuid.New(), Payload: "payload"}

			require.Equal(t, domain.Cause{CommandID: cmd.ID, CorrelationID: cmd.CorrelationID, CausationID: cmd.CausationID}, cmd.Cause())
		})
	})
}
