//go:build unit

package store_test

import (
	"context"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
	"github.com/stretchr/testify/require"
)

func TestMemory(t *testing.T) {
	ctx := context.Background()

	t.Run("append", func(t *testing.T) {
		t.Run("gives each event the next seq of its stream", func(t *testing.T) {
			s := store.NewMemory()

			require.NoError(t, s.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"}, event.AccountCredited{AccountID: "a", Amount: 1}))
			require.NoError(t, s.Append(ctx, "b", 0, event.AccountOpened{AccountID: "b"}))

			records, seq, err := s.Load(ctx, "a")
			require.NoError(t, err)
			require.Equal(t, 2, seq)
			require.Equal(t, []int{1, 2}, []int{records[0].Seq, records[1].Seq})
		})

		t.Run("returns a conflict when the stream is not at the expected seq", func(t *testing.T) {
			s := store.NewMemory()
			require.NoError(t, s.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"}))

			err := s.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"})

			require.ErrorIs(t, err, stream.ErrVersionConflict)
		})
	})
}
