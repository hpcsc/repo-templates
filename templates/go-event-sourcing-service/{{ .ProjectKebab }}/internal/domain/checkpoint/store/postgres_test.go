//go:build integration

package store_test

import (
	"context"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/stretchr/testify/require"
)

func TestPostgresStore(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		t.Run("returns no position for a name without a checkpoint", func(t *testing.T) {
			s := store.NewPostgres(test.NewDBPool(t))

			position, err := s.Get(context.Background(), "test-"+test.NewStringID(t))

			require.NoError(t, err)
			require.Nil(t, position)
		})
	})

	t.Run("set", func(t *testing.T) {
		t.Run("replaces the position of the name", func(t *testing.T) {
			s := store.NewPostgres(test.NewDBPool(t))
			name := "test-" + test.NewStringID(t)
			ctx := context.Background()

			require.NoError(t, s.Set(ctx, name, domain.Position{TransactionID: 750, Sequence: 12}))
			require.NoError(t, s.Set(ctx, name, domain.Position{TransactionID: 760, Sequence: 30}))

			position, err := s.Get(ctx, name)
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 760, Sequence: 30}, position)
		})
	})

	t.Run("set in transaction", func(t *testing.T) {
		t.Run("saves the position only when the transaction commits", func(t *testing.T) {
			pool := test.NewDBPool(t)
			s := store.NewPostgres(pool)
			rolledBackName := "test-" + test.NewStringID(t)
			committedName := "test-" + test.NewStringID(t)
			ctx := context.Background()

			rolledBack, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, s.SetInTx(ctx, rolledBack, rolledBackName, domain.Position{TransactionID: 750, Sequence: 12}))
			require.NoError(t, rolledBack.Rollback(ctx))

			committed, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, s.SetInTx(ctx, committed, committedName, domain.Position{TransactionID: 760, Sequence: 30}))
			require.NoError(t, committed.Commit(ctx))

			rolledBackPosition, err := s.Get(ctx, rolledBackName)
			require.NoError(t, err)
			require.Nil(t, rolledBackPosition)

			committedPosition, err := s.Get(ctx, committedName)
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 760, Sequence: 30}, committedPosition)
		})
	})
}
