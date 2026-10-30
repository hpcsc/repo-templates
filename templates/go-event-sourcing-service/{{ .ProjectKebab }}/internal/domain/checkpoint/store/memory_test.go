//go:build unit

package store_test

import (
	"context"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		t.Run("returns no position for a name without a checkpoint", func(t *testing.T) {
			position, err := store.NewEmptyMemory().Get(context.Background(), "unknown")

			require.NoError(t, err)
			require.Nil(t, position)
		})

		t.Run("returns the position that was set", func(t *testing.T) {
			s := store.NewEmptyMemory()
			require.NoError(t, s.Set(context.Background(), "consumer", domain.Position{TransactionID: 750, Sequence: 12}))

			position, err := s.Get(context.Background(), "consumer")

			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, position)
		})
	})

	t.Run("set", func(t *testing.T) {
		t.Run("refuses a position before the saved one and keeps the saved one", func(t *testing.T) {
			s := store.NewEmptyMemory()
			ctx := context.Background()
			require.NoError(t, s.Set(ctx, "consumer", domain.Position{TransactionID: 760, Sequence: 30}))

			err := s.Set(ctx, "consumer", domain.Position{TransactionID: 750, Sequence: 40})

			require.ErrorIs(t, err, checkpoint.ErrMovedBack)
			position, err := s.Get(ctx, "consumer")
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 760, Sequence: 30}, position)
		})
	})
}
