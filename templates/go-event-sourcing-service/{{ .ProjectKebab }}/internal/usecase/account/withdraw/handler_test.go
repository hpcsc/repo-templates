//go:build unit

package withdraw_test

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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/withdraw"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("saves the withdrawal when the balance covers it", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id), deposited(id, 500))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Withdraw{AccountID: id, Amount: 200}))

			require.NoError(t, err)
			events := eventsOf(t, s, id)
			require.Len(t, events, 3)
			require.Equal(t, &accountEvent.MoneyWithdrawn{AccountID: id, Amount: 200}, events[2].Payload)
		})

		t.Run("rejects an amount more than the balance and saves nothing", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id), deposited(id, 500))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Withdraw{AccountID: id, Amount: 501}))

			require.ErrorIs(t, err, account.ErrInsufficientFunds)
			require.Len(t, eventsOf(t, s, id), 2)
		})
	})
}

func deposited(id uuid.UUID, amount int64) domain.Event {
	return domain.NewEvent(&accountEvent.MoneyDeposited{AccountID: id, Amount: amount})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	withdraw.Register(bus, account.NewRepository(s))
	return bus, s
}

func given(t *testing.T, s event.Stream, id uuid.UUID, history ...domain.Event) {
	require.NoError(t, s.Save(context.Background(), account.StreamID(id), history, 0))
}

func opened(id uuid.UUID) domain.Event {
	return domain.NewEvent(&accountEvent.AccountOpened{AccountID: id, Owner: "alice"})
}

func eventsOf(t *testing.T, s event.Stream, id uuid.UUID) []domain.Event {
	events, err := s.EventsForStream(context.Background(), account.StreamID(id))
	require.NoError(t, err)
	return events
}
