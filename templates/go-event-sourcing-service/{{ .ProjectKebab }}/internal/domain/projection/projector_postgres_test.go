//go:build integration

package projection_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var errRowFailed = errors.New("row failed")

func TestProjectorWithPostgres(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("commits the read model rows and the checkpoint together", func(t *testing.T) {
			pool := test.NewDBPool(t)
			checkpointStore := store.NewPostgres(pool)
			writer := newRowWriter(t, "fail")
			sub := subscription.NewFake().WithEventChannel(closedChannel(
				rowEvent(1, "written"),
				rowEvent(2, "written"),
			))

			err := projection.NewProjector(writer, pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, event.ErrSubscriptionEnded)
			require.Equal(t, int64(2), countRows(t, pool, writer.owner))
			cp, err := checkpointStore.Get(context.Background(), writer.Name())
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 2}, cp)
		})

		t.Run("rolls back the read model rows and the checkpoint together when a handler fails", func(t *testing.T) {
			pool := test.NewDBPool(t)
			checkpointStore := store.NewPostgres(pool)
			writer := newRowWriter(t, "fail")
			sub := subscription.NewFake().WithEventChannel(closedChannel(
				rowEvent(1, "written"),
				rowEvent(2, "fail"),
			))

			err := projection.NewProjector(writer, pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, errRowFailed)
			require.Equal(t, int64(0), countRows(t, pool, writer.owner))
			cp, err := checkpointStore.Get(context.Background(), writer.Name())
			require.NoError(t, err)
			require.Nil(t, cp)
		})
	})
}

type rowWriter struct {
	name   string
	owner  string
	failOn string
}

func newRowWriter(t *testing.T, failOn string) *rowWriter {
	return &rowWriter{
		name:   "test-" + test.NewStringID(t),
		owner:  "test-" + test.NewStringID(t),
		failOn: failOn,
	}
}

func (r *rowWriter) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	if _, err := tx.Exec(ctx,
		"INSERT INTO account_balances (id, owner, balance, version) VALUES ($1, $2, 0, 1)",
		uuid.NewString(), r.owner,
	); err != nil {
		return err
	}
	if evt.Payload.(*fake.SomethingHappened).Value == r.failOn {
		return errRowFailed
	}
	return nil
}

func (r *rowWriter) EventTypes() []string {
	return nil
}

func (r *rowWriter) Name() string {
	return r.name
}

func rowEvent(sequence uint64, value string) domain.Event {
	return domain.Event{Payload: &fake.SomethingHappened{Value: value}, Position: domain.Position{TransactionID: 750, Sequence: sequence}}
}

func countRows(t *testing.T, pool *pgxpool.Pool, owner string) int64 {
	var count int64
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM account_balances WHERE owner = $1", owner).Scan(&count))
	return count
}
