//go:build unit

package open_test

import (
	"context"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/stretchr/testify/require"
)

var now = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)

func TestRunner(t *testing.T) {
	ctx := context.Background()
	clock := func() time.Time { return now }

	t.Run("run", func(t *testing.T) {
		t.Run("records that the account is open for its owner", func(t *testing.T) {
			s := store.NewMemory()

			report, err := open.New(s, clock).Run(ctx, "acc-1", "Ada")

			require.NoError(t, err)
			require.Equal(t, open.Report{AccountID: "acc-1"}, report)
			records, _, err := s.Load(ctx, account.StreamID("acc-1"))
			require.NoError(t, err)
			require.Equal(t, event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: now}, records[0].Event)
		})

		t.Run("records nothing when the account is already open", func(t *testing.T) {
			s := store.NewMemory()
			runner := open.New(s, clock)
			_, err := runner.Run(ctx, "acc-1", "Ada")
			require.NoError(t, err)

			_, err = runner.Run(ctx, "acc-1", "Grace")

			require.NoError(t, err)
			_, seq, err := s.Load(ctx, account.StreamID("acc-1"))
			require.NoError(t, err)
			require.Equal(t, 1, seq)
		})

		t.Run("refuses an account with no owner", func(t *testing.T) {
			_, err := open.New(store.NewMemory(), clock).Run(ctx, "acc-1", "")

			require.ErrorIs(t, err, open.ErrNoOwner)
		})
	})
}
