//go:build unit

package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	t.Run("caused by", func(t *testing.T) {
		t.Run("keeps the command and the chain of its cause", func(t *testing.T) {
			cause := domain.Cause{CommandID: uuid.New(), CorrelationID: uuid.New(), CausationID: uuid.New()}

			evt := domain.Event{}.CausedBy(cause)

			require.Equal(t, domain.Event{
				CommandID: cause.CommandID,
				Metadata:  &domain.EventMetadata{CorrelationID: cause.CorrelationID, CausationID: cause.CausationID},
			}, evt)
		})
	})

	t.Run("caused metadata", func(t *testing.T) {
		t.Run("passes on the correlation ID of the event, and names the event as the cause", func(t *testing.T) {
			id, correlationID := uuid.New(), uuid.New()
			evt := domain.Event{ID: id.String(), Metadata: &domain.EventMetadata{CorrelationID: correlationID, CausationID: uuid.New()}}

			require.Equal(t, domain.EventMetadata{CorrelationID: correlationID, CausationID: id}, evt.CausedMetadata())
		})

		t.Run("starts a chain with the event ID when the event has no correlation ID", func(t *testing.T) {
			id := uuid.New()
			evt := domain.Event{ID: id.String()}

			require.Equal(t, domain.EventMetadata{CorrelationID: id, CausationID: id}, evt.CausedMetadata())
		})
	})
}
