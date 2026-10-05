//go:build integration

package stream_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type versionedThingHappened struct {
	Value string
}

func (versionedThingHappened) EventType() string {
	return "test.versioned-thing-happened"
}

func TestPostgresStream(t *testing.T) {
	t.Run("save", func(t *testing.T) {
		t.Run("appends events with the versions after the expected version", func(t *testing.T) {
			s := newStream(t)
			streamID := newStreamID(t)
			ctx := context.Background()

			require.NoError(t, s.Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "first")}, 0))
			require.NoError(t, s.Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "second")}, 1))

			events, err := s.EventsForStream(ctx, streamID)
			require.NoError(t, err)
			require.Len(t, events, 2)
			require.Equal(t, uint64(1), events[0].Version)
			require.Equal(t, uint64(2), events[1].Version)
			require.Equal(t, "second", events[1].Payload.(*fake.SomethingHappened).Value)
			require.NotEmpty(t, events[0].ID)
		})

		t.Run("keeps the ID that the event already has", func(t *testing.T) {
			s := newStream(t)
			streamID := newStreamID(t)
			evt := fake.NewSomethingHappenedEvent("1", "with-id")
			evt.ID = test.NewStringID(t)

			require.NoError(t, s.Save(context.Background(), streamID, []domain.Event{evt}, 0))

			events, err := s.EventsForStream(context.Background(), streamID)
			require.NoError(t, err)
			require.Equal(t, evt.ID, events[0].ID)
		})

		t.Run("returns a concurrency conflict when the stream is past the expected version", func(t *testing.T) {
			s := newStream(t)
			streamID := newStreamID(t)
			ctx := context.Background()
			require.NoError(t, s.Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "first")}, 0))

			err := s.Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "stale")}, 0)

			require.ErrorIs(t, err, event.ErrConcurrencyConflict)
		})

		t.Run("returns a concurrency conflict when the expected version is past the stream", func(t *testing.T) {
			s := newStream(t)

			err := s.Save(context.Background(), newStreamID(t), []domain.Event{fake.NewSomethingHappenedEvent("1", "ahead")}, 3)

			require.ErrorIs(t, err, event.ErrConcurrencyConflict)
		})

		t.Run("saves the schema version that the registry has for the event type", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := registry.New()
			reg.Register[versionedThingHappened](registry.WithSchemaVersion(3, nil))
			streamID := newStreamID(t)
			ctx := context.Background()

			require.NoError(t, stream.NewPostgres(pool, reg).Save(ctx, streamID, []domain.Event{domain.NewEvent(&versionedThingHappened{Value: "saved"})}, 0))

			var schemaVersion int
			require.NoError(t, pool.QueryRow(ctx, "SELECT schema_version FROM events WHERE stream_id = $1", streamID).Scan(&schemaVersion))
			require.Equal(t, 3, schemaVersion)
		})

		t.Run("saves nothing and returns an unknown type error for a type that the registry does not know", func(t *testing.T) {
			pool := test.NewDBPool(t)
			streamID := newStreamID(t)
			ctx := context.Background()

			err := stream.NewPostgres(pool, registry.New()).Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "unknown")}, 0)

			require.ErrorIs(t, err, registry.ErrUnknownEventType)
			events, err := newStreamOn(pool).EventsForStream(ctx, streamID)
			require.NoError(t, err)
			require.Empty(t, events)
		})
	})

	t.Run("save in transaction", func(t *testing.T) {
		t.Run("appends the events only when the transaction commits", func(t *testing.T) {
			pool := test.NewDBPool(t)
			s := newStreamOn(pool)
			rolledBackID := newStreamID(t)
			committedID := newStreamID(t)
			ctx := context.Background()

			rolledBack, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, s.SaveInTx(ctx, rolledBack, rolledBackID, []domain.Event{fake.NewSomethingHappenedEvent("1", "rolled-back")}, 0))
			require.NoError(t, rolledBack.Rollback(ctx))

			committed, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, s.SaveInTx(ctx, committed, committedID, []domain.Event{fake.NewSomethingHappenedEvent("1", "committed")}, 0))
			require.NoError(t, committed.Commit(ctx))

			rolledBackEvents, err := s.EventsForStream(ctx, rolledBackID)
			require.NoError(t, err)
			require.Empty(t, rolledBackEvents)
			committedEvents, err := s.EventsForStream(ctx, committedID)
			require.NoError(t, err)
			require.Len(t, committedEvents, 1)
		})
	})

	t.Run("change a saved event", func(t *testing.T) {
		t.Run("the table refuses an update", func(t *testing.T) {
			pool := test.NewDBPool(t)
			streamID := newStreamID(t)
			ctx := context.Background()
			require.NoError(t, newStreamOn(pool).Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "saved")}, 0))

			_, err := pool.Exec(ctx, "UPDATE events SET data = '{}' WHERE stream_id = $1", streamID)

			require.ErrorContains(t, err, "events are append-only")
		})

		t.Run("the table refuses a delete", func(t *testing.T) {
			pool := test.NewDBPool(t)
			streamID := newStreamID(t)
			ctx := context.Background()
			require.NoError(t, newStreamOn(pool).Save(ctx, streamID, []domain.Event{fake.NewSomethingHappenedEvent("1", "saved")}, 0))

			_, err := pool.Exec(ctx, "DELETE FROM events WHERE stream_id = $1", streamID)

			require.ErrorContains(t, err, "events are append-only")
		})

		t.Run("the table refuses a truncate", func(t *testing.T) {
			pool := test.NewDBPool(t)
			ctx := context.Background()
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(ctx) }()

			_, err = tx.Exec(ctx, "TRUNCATE events")

			require.ErrorContains(t, err, "events are append-only")
		})
	})

	t.Run("events for stream", func(t *testing.T) {
		t.Run("returns the events in version order, each with its position", func(t *testing.T) {
			s := newStream(t)
			streamID := newStreamID(t)
			ctx := context.Background()
			require.NoError(t, s.Save(ctx, streamID, []domain.Event{
				fake.NewSomethingHappenedEvent("1", "first"),
				fake.NewSomethingHappenedEvent("1", "second"),
			}, 0))

			events, err := s.EventsForStream(ctx, streamID)

			require.NoError(t, err)
			require.Equal(t, "first", events[0].Payload.(*fake.SomethingHappened).Value)
			require.Equal(t, "second", events[1].Payload.(*fake.SomethingHappened).Value)
			require.NotZero(t, events[0].Position.TransactionID)
			require.Less(t, events[0].Position.Sequence, events[1].Position.Sequence)
		})

		t.Run("upcasts an event that has an older schema version", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := registry.New()
			reg.Register[versionedThingHappened](registry.WithSchemaVersion(2, func(_ int, rawPayload []byte) ([]byte, error) {
				var old struct{ Text string }
				if err := json.Unmarshal(rawPayload, &old); err != nil {
					return nil, err
				}
				return json.Marshal(versionedThingHappened{Value: old.Text})
			}))
			streamID := newStreamID(t)
			ctx := context.Background()
			_, err := pool.Exec(ctx,
				"INSERT INTO events (id, stream_id, version, type, schema_version, data) VALUES ($1, $2, 1, $3, 1, $4)",
				uuid.NewString(), streamID, versionedThingHappened{}.EventType(), `{"Text": "from-version-1"}`,
			)
			require.NoError(t, err)

			events, err := stream.NewPostgres(pool, reg).EventsForStream(ctx, streamID)

			require.NoError(t, err)
			require.Equal(t, &versionedThingHappened{Value: "from-version-1"}, events[0].Payload)
		})

		t.Run("returns no events for a stream that does not exist", func(t *testing.T) {
			events, err := newStream(t).EventsForStream(context.Background(), newStreamID(t))

			require.NoError(t, err)
			require.Empty(t, events)
		})
	})
}

func newStream(t *testing.T) event.Stream {
	return newStreamOn(test.NewDBPool(t))
}

func newStreamOn(pool *pgxpool.Pool) event.Stream {
	reg := registry.New()
	reg.Register[fake.SomethingHappened]()
	return stream.NewPostgres(pool, reg)
}

func newStreamID(t *testing.T) string {
	return "test-" + test.NewStringID(t)
}
