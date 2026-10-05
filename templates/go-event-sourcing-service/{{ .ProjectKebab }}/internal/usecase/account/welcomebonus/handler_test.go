//go:build unit

package welcomebonus_test

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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/welcomebonus"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("saves one welcome bonus when the command arrives twice", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id))
			grant := domain.NewCommand(accountCommand.GrantWelcomeBonus{AccountID: id, Amount: welcomebonus.Amount})

			require.NoError(t, bus.Dispatch(context.Background(), grant))
			require.NoError(t, bus.Dispatch(context.Background(), grant))

			events := eventsOf(t, s, id)
			require.Len(t, events, 2)
			require.Equal(t, &accountEvent.WelcomeBonusGranted{AccountID: id, Amount: welcomebonus.Amount}, events[1].Payload)
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	welcomebonus.Register(bus, account.NewRepository(s))
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
