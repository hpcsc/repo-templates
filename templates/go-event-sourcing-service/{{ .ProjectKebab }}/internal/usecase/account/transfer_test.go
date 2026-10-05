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

var transferID = uuid.MustParse("0b8e3d1a-5c47-4e2f-8a6b-9d1c2e3f4a50")

func TestAccountTransfers(t *testing.T) {
	t.Run("debit for transfer", func(t *testing.T) {
		t.Run("records the debit when the balance covers it", func(t *testing.T) {
			a := given(opened(), deposited(500))

			require.NoError(t, a.DebitForTransfer(transferID, 300))

			require.Equal(t, &event.TransferDebited{AccountID: accountID, TransferID: transferID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("records a rejection, not an error, when the balance is too low", func(t *testing.T) {
			a := given(opened(), deposited(100))

			require.NoError(t, a.DebitForTransfer(transferID, 300))

			require.Equal(t, &event.TransferDebitRejected{AccountID: accountID, TransferID: transferID, Amount: 300, Reason: account.ErrInsufficientFunds.Error()}, onlyRecorded(t, a).Payload)
		})

		t.Run("records a rejection when the account is closed", func(t *testing.T) {
			a := given(opened(), deposited(500), closed())

			require.NoError(t, a.DebitForTransfer(transferID, 300))

			require.Equal(t, account.ErrClosed.Error(), onlyRecorded(t, a).Payload.(*event.TransferDebitRejected).Reason)
		})

		t.Run("records nothing when it already decided on the transfer", func(t *testing.T) {
			a := given(opened(), deposited(100), domain.NewEvent(&event.TransferDebitRejected{AccountID: accountID, TransferID: transferID, Amount: 300}))

			require.NoError(t, a.DebitForTransfer(transferID, 300))

			require.Empty(t, a.Recorded())
		})
	})

	t.Run("credit for transfer", func(t *testing.T) {
		t.Run("records the credit", func(t *testing.T) {
			a := given(opened())

			require.NoError(t, a.CreditForTransfer(transferID, 300))

			require.Equal(t, &event.TransferCredited{AccountID: accountID, TransferID: transferID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("records a rejection when the account is closed", func(t *testing.T) {
			a := given(opened(), closed())

			require.NoError(t, a.CreditForTransfer(transferID, 300))

			require.Equal(t, &event.TransferCreditRejected{AccountID: accountID, TransferID: transferID, Amount: 300, Reason: account.ErrClosed.Error()}, onlyRecorded(t, a).Payload)
		})

		t.Run("records nothing when it already credited the transfer", func(t *testing.T) {
			a := given(opened(), domain.NewEvent(&event.TransferCredited{AccountID: accountID, TransferID: transferID, Amount: 300}))

			require.NoError(t, a.CreditForTransfer(transferID, 300))

			require.Empty(t, a.Recorded())
		})
	})

	t.Run("refund transfer", func(t *testing.T) {
		t.Run("records the refund, also on a closed account", func(t *testing.T) {
			a := given(opened(), closed())

			require.NoError(t, a.RefundTransfer(transferID, 300))

			require.Equal(t, &event.TransferRefunded{AccountID: accountID, TransferID: transferID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("records nothing when it already refunded the transfer", func(t *testing.T) {
			a := given(opened(), domain.NewEvent(&event.TransferRefunded{AccountID: accountID, TransferID: transferID, Amount: 300}))

			require.NoError(t, a.RefundTransfer(transferID, 300))

			require.Empty(t, a.Recorded())
		})

		t.Run("rejects an account that is not open", func(t *testing.T) {
			var a account.Account

			require.ErrorIs(t, a.RefundTransfer(transferID, 300), account.ErrNotOpened)
		})
	})
}
