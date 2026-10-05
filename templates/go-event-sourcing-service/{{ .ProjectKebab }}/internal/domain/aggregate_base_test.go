//go:build unit

package domain_test

import (
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/stretchr/testify/require"
)

func TestAggregateBase(t *testing.T) {
	t.Run("record", func(t *testing.T) {
		t.Run("keeps the recorded events in order", func(t *testing.T) {
			var b domain.AggregateBase

			b.Record(fake.NewSomethingHappenedEvent("1", "first"))
			b.Record(fake.NewSomethingHappenedEvent("1", "second"))

			require.Equal(t, []domain.Event{fake.NewSomethingHappenedEvent("1", "first"), fake.NewSomethingHappenedEvent("1", "second")}, b.Recorded())
		})
	})

	t.Run("mark committed", func(t *testing.T) {
		t.Run("moves to the given version and clears the recorded events", func(t *testing.T) {
			var b domain.AggregateBase
			b.Record(fake.NewSomethingHappenedEvent("1", "first"))

			b.MarkCommitted(3)

			require.Equal(t, uint64(3), b.Version())
			require.Empty(t, b.Recorded())
		})
	})
}
