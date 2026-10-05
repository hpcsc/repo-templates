//go:build unit

package credit_test

import (
	"context"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/credit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/stretchr/testify/require"
)

var now = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)

func clock() time.Time { return now }

func openedStore(t *testing.T) *store.Memory {
	s := store.NewMemory()
	_, err := open.New(s, clock).Run(context.Background(), "acc-1", "Ada")
	require.NoError(t, err)
	return s
}

func TestRunner(t *testing.T) {
	ctx := context.Background()

	t.Run("run", func(t *testing.T) {
		t.Run("records the credit", func(t *testing.T) {
			s := openedStore(t)

			_, err := credit.New(s, clock).Run(ctx, "acc-1", 500, "invoice-7")

			require.NoError(t, err)
			records, _, err := s.Load(ctx, account.StreamID("acc-1"))
			require.NoError(t, err)
			require.Equal(t, event.AccountCredited{AccountID: "acc-1", Amount: 500, Reference: "invoice-7", CreditedAt: now}, records[1].Event)
		})

		t.Run("records nothing for a reference that it already credited", func(t *testing.T) {
			s := openedStore(t)
			runner := credit.New(s, clock)
			_, err := runner.Run(ctx, "acc-1", 500, "invoice-7")
			require.NoError(t, err)

			_, err = runner.Run(ctx, "acc-1", 500, "invoice-7")

			require.NoError(t, err)
			_, seq, err := s.Load(ctx, account.StreamID("acc-1"))
			require.NoError(t, err)
			require.Equal(t, 2, seq)
		})

		t.Run("refuses an account that is not open", func(t *testing.T) {
			_, err := credit.New(store.NewMemory(), clock).Run(ctx, "acc-1", 500, "invoice-7")

			require.ErrorIs(t, err, credit.ErrNotOpen)
		})

		t.Run("refuses an amount that is not more than 0", func(t *testing.T) {
			_, err := credit.New(openedStore(t), clock).Run(ctx, "acc-1", 0, "invoice-7")

			require.ErrorIs(t, err, credit.ErrInvalidAmount)
		})
	})
}
