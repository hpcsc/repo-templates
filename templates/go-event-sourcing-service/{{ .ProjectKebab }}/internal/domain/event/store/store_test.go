//go:build unit

package store_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/stretchr/testify/require"
)

func TestOfEvents(t *testing.T) {
	t.Run("load", func(t *testing.T) {
		t.Run("returns a new aggregate at version 0 for a stream that does not exist", func(t *testing.T) {
			s := store.New(stream.NewMemory())

			aggregate, err := s.Load[fake.Aggregate](context.Background(), "unknown")

			require.NoError(t, err)
			require.Equal(t, uint64(0), aggregate.Version())
			require.Empty(t, aggregate.Value)
		})

		t.Run("applies the saved events and sets the version to their count", func(t *testing.T) {
			s := store.New(stream.NewMemory())
			saved := &fake.Aggregate{}
			saved.DoSomething("1", "first", "second")
			require.NoError(t, s.Save(context.Background(), "stream-1", saved, domain.Cause{CommandID: uuid.New()}))

			loaded, err := s.Load[fake.Aggregate](context.Background(), "stream-1")

			require.NoError(t, err)
			require.Equal(t, uint64(2), loaded.Version())
			require.Equal(t, "second", loaded.Value)
		})
	})

	t.Run("save", func(t *testing.T) {
		t.Run("clears the recorded events and moves the version past them", func(t *testing.T) {
			s := store.New(stream.NewMemory())
			aggregate := &fake.Aggregate{}
			aggregate.DoSomething("1", "first", "second")

			require.NoError(t, s.Save(context.Background(), "stream-1", aggregate, domain.Cause{CommandID: uuid.New()}))

			require.Empty(t, aggregate.Recorded())
			require.Equal(t, uint64(2), aggregate.Version())
		})

		t.Run("stamps the command and the chain of the cause on each saved event", func(t *testing.T) {
			memory := stream.NewMemory()
			s := store.New(memory)
			cause := domain.Cause{CommandID: uuid.New(), CorrelationID: uuid.New(), CausationID: uuid.New()}
			aggregate := &fake.Aggregate{}
			aggregate.DoSomething("1", "first", "second")

			require.NoError(t, s.Save(context.Background(), "stream-1", aggregate, cause))

			events, err := memory.EventsForStream(context.Background(), "stream-1")
			require.NoError(t, err)
			chain := &domain.EventMetadata{CorrelationID: cause.CorrelationID, CausationID: cause.CausationID}
			require.Equal(t, []domain.Event{
				{ID: events[0].ID, StreamID: "stream-1", Version: 1, CommandID: cause.CommandID, Payload: events[0].Payload, Metadata: chain},
				{ID: events[1].ID, StreamID: "stream-1", Version: 2, CommandID: cause.CommandID, Payload: events[1].Payload, Metadata: chain},
			}, events)
		})

		t.Run("returns a concurrency conflict when the stream changed after the load", func(t *testing.T) {
			s := store.New(stream.NewMemory())
			first, err := s.Load[fake.Aggregate](context.Background(), "stream-1")
			require.NoError(t, err)
			second, err := s.Load[fake.Aggregate](context.Background(), "stream-1")
			require.NoError(t, err)
			first.DoSomething("1", "first")
			second.DoSomething("1", "second")
			require.NoError(t, s.Save(context.Background(), "stream-1", first, domain.Cause{CommandID: uuid.New()}))

			err = s.Save(context.Background(), "stream-1", second, domain.Cause{CommandID: uuid.New()})

			require.ErrorIs(t, err, event.ErrConcurrencyConflict)
			require.NotEmpty(t, second.Recorded())
		})
	})
}
