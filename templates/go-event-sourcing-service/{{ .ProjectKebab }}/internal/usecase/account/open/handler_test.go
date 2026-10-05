//go:build unit

package open_test

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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/open"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("opens the account in its own stream", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Open{AccountID: id, Owner: "alice"}))

			require.NoError(t, err)
			events := eventsOf(t, s, id)
			require.Len(t, events, 1)
			require.Equal(t, &accountEvent.AccountOpened{AccountID: id, Owner: "alice"}, events[0].Payload)
		})

		t.Run("rejects a second open of the same account and saves nothing more", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Open{AccountID: id, Owner: "bob"}))

			require.ErrorIs(t, err, account.ErrAlreadyOpened)
			require.Len(t, eventsOf(t, s, id), 1)
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	open.Register(bus, account.NewRepository(s))
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
