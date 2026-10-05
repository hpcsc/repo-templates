//go:build integration

package subscription_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type otherThingHappened struct {
	Value string
}

func (otherThingHappened) EventType() string {
	return "test.other-thing-happened"
}

type unparsableThingHappened struct {
	Value string
}

func (unparsableThingHappened) EventType() string {
	return "test.unparsable-thing-happened"
}

type newerThingHappened struct {
	Value string
}

func (newerThingHappened) EventType() string {
	return "test.newer-thing-happened"
}

func TestPostgresSubscription(t *testing.T) {
	somethingHappened := fake.SomethingHappened{}.EventType()

	t.Run("subscribe", func(t *testing.T) {
		t.Run("receives events written after the subscription starts", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)

			save(t, pool, streamID, newEvent(t, "written-after"))

			require.Equal(t, "written-after", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("receives events written before the subscription when no position is given", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			save(t, pool, streamID, newEvent(t, "written-before"))

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)

			require.Equal(t, "written-before", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("receives only events of the given types", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := newRegistry()
			reg.Register[otherThingHappened]()
			sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			save(t, pool, streamID,
				domain.NewEvent(&otherThingHappened{Value: "not-subscribed"}),
				newEvent(t, "subscribed"),
			)

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)

			evt := receiveFrom(t, ctx, events, streamID)
			require.Equal(t, somethingHappened, evt.Type())
			require.Equal(t, "subscribed", value(evt))
		})

		t.Run("receives events of every registered type when no type is given", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			save(t, pool, streamID, newEvent(t, "any-type"))

			events, err := sub.SubscribeToAll(ctx, nil, nil)
			require.NoError(t, err)

			require.Equal(t, "any-type", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("delivers each event with its stream, version and position", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			save(t, pool, streamID, newEvent(t, "event-1"), newEvent(t, "event-2"))

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)

			first := receiveFrom(t, ctx, events, streamID)
			second := receiveFrom(t, ctx, events, streamID)
			require.Equal(t, streamID, first.StreamID)
			require.Equal(t, uint64(1), first.Version)
			require.Equal(t, uint64(2), second.Version)
			require.Equal(t, first.Position.TransactionID, second.Position.TransactionID)
			require.Less(t, first.Position.Sequence, second.Position.Sequence)
		})

		t.Run("receives events after the given position, not the event at it", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			save(t, pool, streamID,
				newEvent(t, "event-1"),
				newEvent(t, "event-2"),
				newEvent(t, "event-3"),
			)

			all, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)
			_ = receiveFrom(t, ctx, all, streamID)
			second := receiveFrom(t, ctx, all, streamID)

			events, err := sub.SubscribeToAll(ctx, &second.Position, []string{somethingHappened})
			require.NoError(t, err)

			require.Equal(t, "event-3", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("skips events of a type that the registry does not know", func(t *testing.T) {
			pool := test.NewDBPool(t)
			unknownType := "test-" + test.NewStringID(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			unknownStreamID := newStreamID(t)
			knownStreamID := newStreamID(t)

			insertRow(t, ctx, pool, unknownStreamID, unknownType, eventData("unknown"))
			save(t, pool, knownStreamID, newEvent(t, "known"))

			events, err := sub.SubscribeToAll(ctx, nil, []string{unknownType, somethingHappened})
			require.NoError(t, err)

			require.Equal(t, "known", value(receiveFrom(t, ctx, events, unknownStreamID, knownStreamID)))
		})

		t.Run("stops when an event of a known type cannot be parsed", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := registry.New()
			reg.Register[unparsableThingHappened]()
			sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)

			insertRow(t, ctx, pool, newStreamID(t), unparsableThingHappened{}.EventType(), `"not an object"`)

			events, err := sub.SubscribeToAll(ctx, nil, []string{unparsableThingHappened{}.EventType()})
			require.NoError(t, err)

			select {
			case _, open := <-events:
				require.False(t, open, "the subscription must stop instead of delivering or skipping the event")
			case <-ctx.Done():
				require.Fail(t, "the subscription skipped the event and kept running")
			}
		})

		t.Run("stops when an event has a newer schema version than the registry knows", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := registry.New()
			reg.Register[newerThingHappened]()
			sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)

			_, err := pool.Exec(ctx,
				"INSERT INTO events (id, stream_id, version, type, schema_version, data) VALUES ($1, $2, 1, $3, 2, $4)",
				uuid.NewString(), newStreamID(t), newerThingHappened{}.EventType(), `{"Value": "from-a-newer-build"}`,
			)
			require.NoError(t, err)

			events, err := sub.SubscribeToAll(ctx, nil, []string{newerThingHappened{}.EventType()})
			require.NoError(t, err)

			select {
			case _, open := <-events:
				require.False(t, open, "the subscription must stop instead of delivering or skipping the event")
			case <-ctx.Done():
				require.Fail(t, "the subscription skipped the event and kept running")
			}
		})

		t.Run("holds back a committed event until an older open transaction ends", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger(),
				subscription.WithPollInterval(50*time.Millisecond),
			)
			ctx := newContext(t, 5*time.Second)
			olderStreamID := newStreamID(t)
			newerStreamID := newStreamID(t)

			older, err := pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = older.Rollback(context.Background()) }()
			insertRow(t, ctx, older, olderStreamID, somethingHappened, eventData("from-older-transaction"))

			save(t, pool, newerStreamID, newEvent(t, "from-newer-transaction"))

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)

			requireNothingFrom(t, events, 500*time.Millisecond, olderStreamID, newerStreamID)

			require.NoError(t, older.Commit(ctx))

			require.Equal(t, "from-older-transaction", value(receiveFrom(t, ctx, events, olderStreamID, newerStreamID)))
			require.Equal(t, "from-newer-transaction", value(receiveFrom(t, ctx, events, olderStreamID, newerStreamID)))
		})

		t.Run("reads at once when a notification arrives, before the poll interval ends", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger(),
				subscription.WithPollInterval(time.Hour),
			)
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)
			time.Sleep(300 * time.Millisecond)

			save(t, pool, streamID, newEvent(t, "notified"))

			require.Equal(t, "notified", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("reads on the poll interval when no notification arrives", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger(),
				subscription.WithPollInterval(200*time.Millisecond),
			)
			ctx := newContext(t, 5*time.Second)
			streamID := newStreamID(t)

			events, err := sub.SubscribeToAll(ctx, nil, []string{somethingHappened})
			require.NoError(t, err)
			time.Sleep(300 * time.Millisecond)

			insertRow(t, ctx, pool, streamID, somethingHappened, eventData("not-notified"))

			require.Equal(t, "not-notified", value(receiveFrom(t, ctx, events, streamID)))
		})

		t.Run("keeps retrying while the database is unreachable", func(t *testing.T) {
			pool, err := pgxpool.New(context.Background(), "postgres://user:password@127.0.0.1:1/events?sslmode=disable&connect_timeout=1")
			require.NoError(t, err)
			defer pool.Close()
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			events, err := sub.SubscribeToAll(ctx, nil, nil)
			require.NoError(t, err)

			select {
			case <-events:
				require.Fail(t, "the subscription stopped while the database was unreachable")
			case <-time.After(2 * time.Second):
			}

			cancel()

			select {
			case _, open := <-events:
				require.False(t, open)
			case <-time.After(5 * time.Second):
				require.Fail(t, "the subscription did not stop after the context ended")
			}
		})
	})

	t.Run("last position", func(t *testing.T) {
		t.Run("a subscription after the last position receives only events written after it", func(t *testing.T) {
			pool := test.NewDBPool(t)
			sub := subscription.NewPostgres(pool, newRegistry(), test.DiscardLogger())
			ctx := newContext(t, 5*time.Second)
			beforeStreamID := newStreamID(t)
			afterStreamID := newStreamID(t)

			save(t, pool, beforeStreamID, newEvent(t, "written-before"))

			last, err := sub.LastPosition(ctx)
			require.NoError(t, err)
			require.NotNil(t, last)

			events, err := sub.SubscribeToAll(ctx, last, []string{somethingHappened})
			require.NoError(t, err)

			save(t, pool, afterStreamID, newEvent(t, "written-after"))

			require.Equal(t, "written-after", value(receiveFrom(t, ctx, events, beforeStreamID, afterStreamID)))
		})
	})
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func newContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

func newStreamID(t *testing.T) string {
	return "test-" + test.NewStringID(t)
}

func newRegistry() *registry.OfEvents {
	reg := registry.New()
	reg.Register[fake.SomethingHappened]()
	return reg
}

func newEvent(t *testing.T, value string) domain.Event {
	return fake.NewSomethingHappenedEvent(test.NewStringID(t), value)
}

func eventData(value string) string {
	return fmt.Sprintf(`{"ID": %q, "Value": %q}`, uuid.NewString(), value)
}

func save(t *testing.T, pool *pgxpool.Pool, streamID string, events ...domain.Event) {
	reg := newRegistry()
	reg.Register[otherThingHappened]()
	require.NoError(t, stream.NewPostgres(pool, reg).Save(context.Background(), streamID, events, 0))
}

func insertRow(t *testing.T, ctx context.Context, db executor, streamID string, eventType string, data string) {
	_, err := db.Exec(ctx,
		"INSERT INTO events (id, stream_id, version, type, schema_version, data) VALUES ($1, $2, 1, $3, 1, $4)",
		uuid.NewString(), streamID, eventType, data,
	)
	require.NoError(t, err)
}

func receiveFrom(t *testing.T, ctx context.Context, events <-chan domain.Event, streamIDs ...string) domain.Event {
	for {
		select {
		case evt, open := <-events:
			require.True(t, open, "the subscription stopped")
			if slices.Contains(streamIDs, evt.StreamID) {
				return evt
			}
		case <-ctx.Done():
			require.Fail(t, "timeout waiting for event")
			return domain.Event{}
		}
	}
}

func requireNothingFrom(t *testing.T, events <-chan domain.Event, wait time.Duration, streamIDs ...string) {
	timeout := time.After(wait)
	for {
		select {
		case evt, open := <-events:
			require.True(t, open, "the subscription stopped")
			require.False(t, slices.Contains(streamIDs, evt.StreamID), "an event of stream %s arrived", evt.StreamID)
		case <-timeout:
			return
		}
	}
}

func value(evt domain.Event) string {
	return evt.Payload.(*fake.SomethingHappened).Value
}
