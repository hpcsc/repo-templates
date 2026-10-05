//go:build unit

package transfers_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/transfers"
	"github.com/stretchr/testify/require"
)

func TestHandlers(t *testing.T) {
	t.Run("debit", func(t *testing.T) {
		t.Run("saves the debit once when the request arrives twice", func(t *testing.T) {
			bus, s := newBus()
			id, transferID := uuid.New(), uuid.New()
			given(t, s, id, opened(id), deposited(id, 500))
			debit := domain.NewCommand(accountCommand.DebitForTransfer{AccountID: id, TransferID: transferID, Amount: 300})

			require.NoError(t, bus.Dispatch(context.Background(), debit))
			require.NoError(t, bus.Dispatch(context.Background(), debit))

			events := eventsOf(t, s, id)
			require.Len(t, events, 3)
			require.Equal(t, &accountEvent.TransferDebited{AccountID: id, TransferID: transferID, Amount: 300}, events[2].Payload)
		})
	})

	t.Run("credit", func(t *testing.T) {
		t.Run("saves a rejection when the account is closed", func(t *testing.T) {
			bus, s := newBus()
			id, transferID := uuid.New(), uuid.New()
			given(t, s, id, opened(id), domain.NewEvent(&accountEvent.AccountClosed{AccountID: id}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.CreditForTransfer{AccountID: id, TransferID: transferID, Amount: 300}))

			require.NoError(t, err)
			events := eventsOf(t, s, id)
			require.Len(t, events, 3)
			require.Equal(t, account.ErrClosed.Error(), events[2].Payload.(*accountEvent.TransferCreditRejected).Reason)
		})
	})

	t.Run("refund", func(t *testing.T) {
		t.Run("saves the refund", func(t *testing.T) {
			bus, s := newBus()
			id, transferID := uuid.New(), uuid.New()
			given(t, s, id, opened(id))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.RefundTransfer{AccountID: id, TransferID: transferID, Amount: 300}))

			require.NoError(t, err)
			events := eventsOf(t, s, id)
			require.Equal(t, &accountEvent.TransferRefunded{AccountID: id, TransferID: transferID, Amount: 300}, events[1].Payload)
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	transfers.Register(bus, account.NewRepository(s))
	return bus, s
}

func given(t *testing.T, s event.Stream, id uuid.UUID, history ...domain.Event) {
	require.NoError(t, s.Save(context.Background(), account.StreamID(id), history, 0))
}

func opened(id uuid.UUID) domain.Event {
	return domain.NewEvent(&accountEvent.AccountOpened{AccountID: id, Owner: "alice"})
}

func deposited(id uuid.UUID, amount int64) domain.Event {
	return domain.NewEvent(&accountEvent.MoneyDeposited{AccountID: id, Amount: amount})
}

func eventsOf(t *testing.T, s event.Stream, id uuid.UUID) []domain.Event {
	events, err := s.EventsForStream(context.Background(), account.StreamID(id))
	require.NoError(t, err)
	return events
}
