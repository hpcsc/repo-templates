//go:build unit

package account_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	"github.com/stretchr/testify/require"
)

func TestAccount(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Run("opens the account for the owner", func(t *testing.T) {
			var a account.Account
			id := uuid.New()

			require.NoError(t, a.Open(id, "alice"))

			require.Equal(t, &event.AccountOpened{AccountID: id, Owner: "alice"}, onlyRecorded(t, &a).Payload)
			require.Equal(t, id, a.ID())
		})

		t.Run("rejects a second open of the same account", func(t *testing.T) {
			a := given(opened())

			require.ErrorIs(t, a.Open(uuid.New(), "bob"), account.ErrAlreadyOpened)
		})

		t.Run("rejects an empty owner", func(t *testing.T) {
			var a account.Account

			require.Error(t, a.Open(uuid.New(), ""))
			require.Empty(t, a.Recorded())
		})
	})

	t.Run("deposit", func(t *testing.T) {
		t.Run("records the deposit", func(t *testing.T) {
			a := given(opened())

			require.NoError(t, a.Deposit(500))

			require.Equal(t, int64(500), onlyRecorded(t, a).Payload.(*event.MoneyDeposited).Amount)
		})

		t.Run("rejects an account that is not open", func(t *testing.T) {
			var a account.Account

			require.ErrorIs(t, a.Deposit(500), account.ErrNotOpened)
		})

		t.Run("rejects an amount of 0", func(t *testing.T) {
			a := given(opened())

			require.ErrorIs(t, a.Deposit(0), account.ErrInvalidAmount)
		})
	})

	t.Run("withdraw", func(t *testing.T) {
		t.Run("records the withdrawal when the balance covers it", func(t *testing.T) {
			a := given(opened(), deposited(500))

			require.NoError(t, a.Withdraw(500))

			require.Equal(t, int64(500), onlyRecorded(t, a).Payload.(*event.MoneyWithdrawn).Amount)
		})

		t.Run("rejects an amount more than the balance", func(t *testing.T) {
			a := given(opened(), deposited(500))

			require.ErrorIs(t, a.Withdraw(501), account.ErrInsufficientFunds)
			require.Empty(t, a.Recorded())
		})
	})

	t.Run("close", func(t *testing.T) {
		t.Run("closes an open account", func(t *testing.T) {
			a := given(opened())

			require.NoError(t, a.Close())

			require.Equal(t, &event.AccountClosed{AccountID: accountID}, onlyRecorded(t, a).Payload)
		})

		t.Run("rejects an account that is not open", func(t *testing.T) {
			var a account.Account

			require.ErrorIs(t, a.Close(), account.ErrNotOpened)
		})

		t.Run("rejects a second close", func(t *testing.T) {
			a := given(opened(), closed())

			require.ErrorIs(t, a.Close(), account.ErrClosed)
		})

		t.Run("rejects deposits and withdrawals on a closed account", func(t *testing.T) {
			a := given(opened(), deposited(500), closed())

			require.ErrorIs(t, a.Deposit(100), account.ErrClosed)
			require.ErrorIs(t, a.Withdraw(100), account.ErrClosed)
			require.Empty(t, a.Recorded())
		})
	})

	t.Run("grant welcome bonus", func(t *testing.T) {
		t.Run("grants the bonus", func(t *testing.T) {
			a := given(opened())

			require.NoError(t, a.GrantWelcomeBonus(1000))

			require.Equal(t, int64(1000), onlyRecorded(t, a).Payload.(*event.WelcomeBonusGranted).Amount)
		})

		t.Run("records nothing for a closed account", func(t *testing.T) {
			a := given(opened(), closed())

			require.NoError(t, a.GrantWelcomeBonus(1000))

			require.Empty(t, a.Recorded())
		})

		t.Run("records nothing when the account already has its bonus", func(t *testing.T) {
			a := given(opened(), bonusGranted(1000))

			require.NoError(t, a.GrantWelcomeBonus(1000))

			require.Empty(t, a.Recorded())
		})
	})
}

var accountID = uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71")

func given(history ...domain.Event) *account.Account {
	var a account.Account
	for _, e := range history {
		a.Apply(e)
	}
	a.MarkCommitted(uint64(len(history)))
	return &a
}

func opened() domain.Event {
	return domain.NewEvent(&event.AccountOpened{AccountID: accountID, Owner: "alice"})
}

func deposited(amount int64) domain.Event {
	return domain.NewEvent(&event.MoneyDeposited{AccountID: accountID, Amount: amount})
}

func bonusGranted(amount int64) domain.Event {
	return domain.NewEvent(&event.WelcomeBonusGranted{AccountID: accountID, Amount: amount})
}

func closed() domain.Event {
	return domain.NewEvent(&event.AccountClosed{AccountID: accountID})
}

func onlyRecorded(t *testing.T, a *account.Account) domain.Event {
	recorded := a.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}
