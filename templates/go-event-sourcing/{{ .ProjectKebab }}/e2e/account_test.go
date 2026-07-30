//go:build e2e

package e2e_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccount(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)

	t.Run("open then credit", func(t *testing.T) {
		t.Run("survives a round trip through postgres", func(t *testing.T) {
			application := newApp(t, at)
			id := accountID(t)

			_, err := application.Open.Run(ctx, id, "Ada")
			require.NoError(t, err)
			_, err = application.Credit.Run(ctx, id, 500, "invoice-7")
			require.NoError(t, err)
			_, err = application.Credit.Run(ctx, id, 250, "invoice-8")
			require.NoError(t, err)

			// A separate app reads through a fresh connection, so this proves
			// the events were decoded from storage rather than remembered.
			view, found, err := newApp(t, at).Show.Run(ctx, id)

			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, "Ada", view.Owner)
			require.Equal(t, int64(750), view.Balance)
			require.Len(t, view.Credits, 2)
			require.Equal(t, at, view.OpenedAt)
		})

		t.Run("applies a repeated credit reference only once", func(t *testing.T) {
			application := newApp(t, at)
			id := accountID(t)

			_, err := application.Open.Run(ctx, id, "Ada")
			require.NoError(t, err)
			_, err = application.Credit.Run(ctx, id, 500, "invoice-7")
			require.NoError(t, err)
			_, err = application.Credit.Run(ctx, id, 500, "invoice-7")
			require.NoError(t, err)

			view, _, err := application.Show.Run(ctx, id)

			require.NoError(t, err)
			require.Equal(t, int64(500), view.Balance)
			require.Len(t, view.Credits, 1)
		})

		t.Run("reports an account nobody opened as missing", func(t *testing.T) {
			_, found, err := newApp(t, at).Show.Run(ctx, accountID(t))

			require.NoError(t, err)
			require.False(t, found)
		})
	})

	t.Run("concurrent writers", func(t *testing.T) {
		t.Run("serialise onto the stream without losing a credit", func(t *testing.T) {
			application := newApp(t, at)
			id := accountID(t)

			_, err := application.Open.Run(ctx, id, "Ada")
			require.NoError(t, err)

			// The unique constraint on (stream_id, version) makes one writer
			// lose; the repository's retry is what stops that being an error.
			const writers = 8
			var wg sync.WaitGroup
			errs := make([]error, writers)
			for i := range writers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, errs[i] = application.Credit.Run(ctx, id, 100, referenceFor(i))
				}()
			}
			wg.Wait()

			for i, err := range errs {
				require.NoError(t, err, "writer %d", i)
			}

			view, _, err := application.Show.Run(ctx, id)
			require.NoError(t, err)
			require.Equal(t, int64(writers*100), view.Balance)
			require.Len(t, view.Credits, writers)
		})
	})
}

func referenceFor(i int) string {
	return string(rune('a'+i)) + "-credit"
}
