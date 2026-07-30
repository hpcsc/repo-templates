//go:build unit

package aggregate_test

import (
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/aggregate"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/stretchr/testify/require"
)

func rebuild(history ...es.Event) *aggregate.Account {
	a := aggregate.NewAccount()
	for _, e := range history {
		a.Apply(e)
	}
	return a
}

func TestAccount(t *testing.T) {
	at := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)

	t.Run("open account", func(t *testing.T) {
		t.Run("opens a fresh account", func(t *testing.T) {
			events, err := aggregate.NewAccount().OpenAccount(command.OpenAccount{
				AccountID: "acc-1", Owner: "Ada", OpenedAt: at,
			})

			require.NoError(t, err)
			require.Equal(t, []es.Event{event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: at}}, events)
		})

		t.Run("decides nothing when the account is already open", func(t *testing.T) {
			a := rebuild(event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: at})

			events, err := a.OpenAccount(command.OpenAccount{AccountID: "acc-1", Owner: "Ada", OpenedAt: at})

			require.NoError(t, err)
			require.Empty(t, events)
		})

		t.Run("rejects an account with no owner", func(t *testing.T) {
			events, err := aggregate.NewAccount().OpenAccount(command.OpenAccount{AccountID: "acc-1", OpenedAt: at})

			require.ErrorContains(t, err, "owner")
			require.Nil(t, events)
		})
	})

	t.Run("credit account", func(t *testing.T) {
		t.Run("credits an open account", func(t *testing.T) {
			a := rebuild(event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: at})

			events, err := a.CreditAccount(command.CreditAccount{
				AccountID: "acc-1", Amount: 500, Reference: "invoice-7", CreditedAt: at,
			})

			require.NoError(t, err)
			require.Len(t, events, 1)
			require.Equal(t, int64(500), events[0].(event.AccountCredited).Amount)
		})

		t.Run("rejects a credit to an account that was never opened", func(t *testing.T) {
			events, err := aggregate.NewAccount().CreditAccount(command.CreditAccount{
				AccountID: "acc-1", Amount: 500, Reference: "invoice-7", CreditedAt: at,
			})

			require.ErrorContains(t, err, "not open")
			require.Nil(t, events)
		})

		t.Run("rejects a credit that is not positive", func(t *testing.T) {
			a := rebuild(event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: at})

			events, err := a.CreditAccount(command.CreditAccount{
				AccountID: "acc-1", Amount: 0, Reference: "invoice-7", CreditedAt: at,
			})

			require.ErrorContains(t, err, "positive")
			require.Nil(t, events)
		})

		t.Run("ignores a credit whose reference was already applied", func(t *testing.T) {
			a := rebuild(
				event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: at},
				event.AccountCredited{AccountID: "acc-1", Amount: 500, Reference: "invoice-7", CreditedAt: at},
			)

			events, err := a.CreditAccount(command.CreditAccount{
				AccountID: "acc-1", Amount: 500, Reference: "invoice-7", CreditedAt: at,
			})

			require.NoError(t, err)
			require.Empty(t, events)
			require.Equal(t, int64(500), a.Balance())
		})
	})
}
