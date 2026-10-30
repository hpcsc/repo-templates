//go:build unit

package account_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account/event"
	"github.com/stretchr/testify/require"
)

var paymentID = uuid.MustParse("3c9f1e27-8a4d-4b6e-9f02-7d5a1c8e3b64")

func TestAccountHolds(t *testing.T) {
	t.Run("place hold", func(t *testing.T) {
		t.Run("records the hold when the available balance covers it", func(t *testing.T) {
			a := given(opened(), deposited(500))

			require.NoError(t, a.PlaceHold(paymentID, 300))

			require.Equal(t, &event.HoldPlaced{AccountID: accountID, PaymentID: paymentID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("records a rejection, not an error, when the held amounts leave too little", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(uuid.New(), 300))

			require.NoError(t, a.PlaceHold(paymentID, 300))

			require.Equal(t, &event.HoldRejected{AccountID: accountID, PaymentID: paymentID, Amount: 300, Reason: account.ErrInsufficientFunds.Error()}, onlyRecorded(t, a).Payload)
		})

		t.Run("records a rejection when the account is closed", func(t *testing.T) {
			a := given(opened(), deposited(500), closed())

			require.NoError(t, a.PlaceHold(paymentID, 300))

			require.Equal(t, account.ErrClosed.Error(), onlyRecorded(t, a).Payload.(*event.HoldRejected).Reason)
		})

		t.Run("records nothing when it already decided on the payment", func(t *testing.T) {
			a := given(opened(), deposited(100), domain.NewEvent(&event.HoldRejected{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.NoError(t, a.PlaceHold(paymentID, 300))

			require.Empty(t, a.Recorded())
		})

		t.Run("rejects a hold on an account that is not open", func(t *testing.T) {
			var a account.Account

			require.ErrorIs(t, a.PlaceHold(paymentID, 300), account.ErrNotOpened)
		})
	})

	t.Run("release hold", func(t *testing.T) {
		t.Run("records the release of the held amount", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300))

			require.NoError(t, a.ReleaseHold(paymentID))

			require.Equal(t, &event.HoldReleased{AccountID: accountID, PaymentID: paymentID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("makes the released amount available again", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300), domain.NewEvent(&event.HoldReleased{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.NoError(t, a.Withdraw(500))

			require.Equal(t, &event.MoneyWithdrawn{AccountID: accountID, Amount: 500}, onlyRecorded(t, a).Payload)
		})

		t.Run("records nothing for a payment with no hold", func(t *testing.T) {
			a := given(opened(), deposited(100), domain.NewEvent(&event.HoldRejected{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.NoError(t, a.ReleaseHold(paymentID))

			require.Empty(t, a.Recorded())
		})

		t.Run("records nothing when it already released the hold", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300), domain.NewEvent(&event.HoldReleased{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.NoError(t, a.ReleaseHold(paymentID))

			require.Empty(t, a.Recorded())
		})
	})

	t.Run("capture hold", func(t *testing.T) {
		t.Run("records the capture of the held amount", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300))

			require.NoError(t, a.CaptureHold(paymentID))

			require.Equal(t, &event.HoldCaptured{AccountID: accountID, PaymentID: paymentID, Amount: 300}, onlyRecorded(t, a).Payload)
		})

		t.Run("records nothing when it already captured the hold", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300), domain.NewEvent(&event.HoldCaptured{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.NoError(t, a.CaptureHold(paymentID))

			require.Empty(t, a.Recorded())
		})

		t.Run("rejects a capture for a payment with no hold", func(t *testing.T) {
			a := given(opened(), deposited(500))

			require.ErrorIs(t, a.CaptureHold(paymentID), account.ErrNoHold)
		})

		t.Run("rejects a capture of a released hold", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300), domain.NewEvent(&event.HoldReleased{AccountID: accountID, PaymentID: paymentID, Amount: 300}))

			require.ErrorIs(t, a.CaptureHold(paymentID), account.ErrNoHold)
		})
	})

	t.Run("withdraw", func(t *testing.T) {
		t.Run("rejects a withdrawal of held money", func(t *testing.T) {
			a := given(opened(), deposited(500), holdPlaced(paymentID, 300))

			require.ErrorIs(t, a.Withdraw(300), account.ErrInsufficientFunds)
		})
	})
}

func holdPlaced(paymentID uuid.UUID, amount int64) domain.Event {
	return domain.NewEvent(&event.HoldPlaced{AccountID: accountID, PaymentID: paymentID, Amount: amount})
}
