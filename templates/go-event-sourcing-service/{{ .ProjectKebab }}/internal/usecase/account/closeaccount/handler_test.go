//go:build unit

package closeaccount_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/closeaccount"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("saves the close after the events that the account already has", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Close{AccountID: id}))

			require.NoError(t, err)
			events := eventsOf(t, s, id)
			require.Len(t, events, 2)
			require.Equal(t, &accountEvent.AccountClosed{AccountID: id}, events[1].Payload)
		})

		t.Run("rejects a second close and saves nothing more", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id), domain.NewEvent(&accountEvent.AccountClosed{AccountID: id}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Close{AccountID: id}))

			require.ErrorIs(t, err, account.ErrClosed)
			require.Len(t, eventsOf(t, s, id), 2)
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	closeaccount.Register(bus, account.NewRepository(s))
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
