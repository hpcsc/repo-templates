//go:build unit

package transfer_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/transfer"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/transfer/event"
	"github.com/stretchr/testify/require"
)

func TestTransfer(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("starts the transfer between two accounts", func(t *testing.T) {
			var tr transfer.Transfer
			id, from, to := uuid.New(), uuid.New(), uuid.New()

			require.NoError(t, tr.Start(id, from, to, 300))

			require.Equal(t, []domain.Event{domain.NewEvent(&event.TransferStarted{TransferID: id, From: from, To: to, Amount: 300})}, tr.Recorded())
			require.Equal(t, id, tr.ID())
		})

		t.Run("rejects a transfer to the same account", func(t *testing.T) {
			var tr transfer.Transfer
			account := uuid.New()

			require.ErrorIs(t, tr.Start(uuid.New(), account, account, 300), transfer.ErrSameAccount)
		})

		t.Run("rejects an amount of 0", func(t *testing.T) {
			var tr transfer.Transfer

			require.ErrorIs(t, tr.Start(uuid.New(), uuid.New(), uuid.New(), 0), transfer.ErrInvalidAmount)
		})

		t.Run("rejects a second start", func(t *testing.T) {
			var tr transfer.Transfer
			id := uuid.New()
			require.NoError(t, tr.Start(id, uuid.New(), uuid.New(), 300))

			require.ErrorIs(t, tr.Start(id, uuid.New(), uuid.New(), 300), transfer.ErrAlreadyStarted)
		})
	})
}
