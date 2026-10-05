//go:build unit

package start_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer"
	transferCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/command"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/start"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("starts the transfer in its own stream when both accounts exist", func(t *testing.T) {
			bus, s := newBus()
			id, from, to := uuid.New(), uuid.New(), uuid.New()
			openAccount(t, s, from)
			openAccount(t, s, to)

			err := bus.Dispatch(context.Background(), domain.NewCommand(transferCommand.Start{TransferID: id, From: from, To: to, Amount: 300}))

			require.NoError(t, err)
			events, err := s.EventsForStream(context.Background(), transfer.StreamID(id))
			require.NoError(t, err)
			require.Equal(t, []domain.Event{
				{
					ID:       events[0].ID,
					StreamID: transfer.StreamID(id),
					Version:  1,
					Payload:  &transferEvent.TransferStarted{TransferID: id, From: from, To: to, Amount: 300},
					Metadata: events[0].Metadata,
				},
			}, events)
		})

		t.Run("rejects a transfer to an account that does not exist", func(t *testing.T) {
			bus, s := newBus()
			id, from, to := uuid.New(), uuid.New(), uuid.New()
			openAccount(t, s, from)

			err := bus.Dispatch(context.Background(), domain.NewCommand(transferCommand.Start{TransferID: id, From: from, To: to, Amount: 300}))

			require.ErrorIs(t, err, transfer.ErrUnknownAccount)
			events, err := s.EventsForStream(context.Background(), transfer.StreamID(id))
			require.NoError(t, err)
			require.Empty(t, events)
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	start.Register(bus, transfer.NewRepository(s), account.NewRepository(s))
	return bus, s
}

func openAccount(t *testing.T, s event.Stream, id uuid.UUID) {
	opened := domain.NewEvent(&accountEvent.AccountOpened{AccountID: id, Owner: "alice"})
	require.NoError(t, s.Save(context.Background(), account.StreamID(id), []domain.Event{opened}, 0))
}
