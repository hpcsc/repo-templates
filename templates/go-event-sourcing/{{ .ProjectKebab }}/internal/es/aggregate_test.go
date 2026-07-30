//go:build unit

package es_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es/store"
	"github.com/stretchr/testify/require"
)

type recorded struct {
	Value string
}

func (recorded) EventName() string { return "es_test.recorded" }

func init() { es.Register[recorded]() }

type counter struct {
	seen []string
}

func (c *counter) Apply(event es.Event) {
	if e, ok := event.(recorded); ok {
		c.seen = append(c.seen, e.Value)
	}
}

// conflictingStore rejects the first n appends the way a concurrent writer
// would, then behaves normally.
type conflictingStore struct {
	es.Store
	conflicts int
	appends   int
}

func (s *conflictingStore) Append(ctx context.Context, streamID string, version int, events []es.Event) error {
	s.appends++
	if s.conflicts > 0 {
		s.conflicts--
		return fmt.Errorf("appending to stream %q: %w", streamID, es.ErrVersionConflict)
	}
	return s.Store.Append(ctx, streamID, version, events)
}

func record(value string) func(*counter) ([]es.Event, error) {
	return func(*counter) ([]es.Event, error) {
		return []es.Event{recorded{Value: value}}, nil
	}
}

func TestRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("update", func(t *testing.T) {
		t.Run("appends the decided events to the stream", func(t *testing.T) {
			eventStore := store.NewMemory()
			repo := es.NewRepository[counter](eventStore, 1)

			require.NoError(t, repo.Update(ctx, "stream-1", record("first")))

			history, version, err := eventStore.Load(ctx, "stream-1")
			require.NoError(t, err)
			require.Equal(t, 1, version)
			require.Equal(t, []es.Event{recorded{Value: "first"}}, history)
		})

		t.Run("rebuilds prior state before deciding", func(t *testing.T) {
			eventStore := store.NewMemory()
			repo := es.NewRepository[counter](eventStore, 1)
			require.NoError(t, repo.Update(ctx, "stream-1", record("first")))

			var seen []string
			require.NoError(t, repo.Update(ctx, "stream-1", func(c *counter) ([]es.Event, error) {
				seen = c.seen
				return nil, nil
			}))

			require.Equal(t, []string{"first"}, seen)
		})

		t.Run("appends nothing when the decision yields no events", func(t *testing.T) {
			eventStore := store.NewMemory()
			repo := es.NewRepository[counter](eventStore, 1)

			require.NoError(t, repo.Update(ctx, "stream-1", func(*counter) ([]es.Event, error) {
				return nil, nil
			}))

			_, version, err := eventStore.Load(ctx, "stream-1")
			require.NoError(t, err)
			require.Equal(t, 0, version)
		})

		t.Run("retries when a concurrent append moved the stream underneath it", func(t *testing.T) {
			eventStore := &conflictingStore{Store: store.NewMemory(), conflicts: 2}
			repo := es.NewRepository[counter](eventStore, 3)

			require.NoError(t, repo.Update(ctx, "stream-1", record("first")))
			require.Equal(t, 3, eventStore.appends)
		})

		t.Run("gives up once the attempts are exhausted", func(t *testing.T) {
			eventStore := &conflictingStore{Store: store.NewMemory(), conflicts: 5}
			repo := es.NewRepository[counter](eventStore, 3)

			err := repo.Update(ctx, "stream-1", record("first"))

			require.True(t, errors.Is(err, es.ErrVersionConflict))
			require.Equal(t, 3, eventStore.appends)
		})

		t.Run("surfaces a decision error without appending", func(t *testing.T) {
			eventStore := &conflictingStore{Store: store.NewMemory()}
			repo := es.NewRepository[counter](eventStore, 3)
			rejected := errors.New("rejected by the aggregate")

			err := repo.Update(ctx, "stream-1", func(*counter) ([]es.Event, error) {
				return nil, rejected
			})

			require.True(t, errors.Is(err, rejected))
			require.Zero(t, eventStore.appends)
		})
	})
}
