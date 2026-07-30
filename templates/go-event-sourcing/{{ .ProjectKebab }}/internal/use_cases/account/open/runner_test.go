//go:build unit

package open_test

import (
	"context"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/show"
	"github.com/stretchr/testify/require"
)

func fixedClock(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func TestRunner(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)

	t.Run("run", func(t *testing.T) {
		t.Run("records the opened account on its own stream", func(t *testing.T) {
			eventStore := store.NewMemory()

			report, err := open.New(eventStore, fixedClock(at)).Run(ctx, "acc-1", "Ada")

			require.NoError(t, err)
			require.Equal(t, open.Report{AccountID: "acc-1"}, report)

			view, found, err := show.New(eventStore).Run(ctx, "acc-1")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, "Ada", view.Owner)
			require.Equal(t, at, view.OpenedAt)
		})

		t.Run("is safe to repeat", func(t *testing.T) {
			eventStore := store.NewMemory()
			runner := open.New(eventStore, fixedClock(at))

			_, err := runner.Run(ctx, "acc-1", "Ada")
			require.NoError(t, err)
			_, err = runner.Run(ctx, "acc-1", "Ada")
			require.NoError(t, err)

			_, version, err := eventStore.Load(ctx, "acc-1")
			require.NoError(t, err)
			require.Equal(t, 1, version)
		})

		t.Run("surfaces a rejected command as an error", func(t *testing.T) {
			_, err := open.New(store.NewMemory(), fixedClock(at)).Run(ctx, "acc-1", "")

			require.ErrorContains(t, err, "owner")
		})
	})
}
