//go:build unit

package welcomebonus_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/welcomebonus"
	"github.com/stretchr/testify/require"
)

func TestReaction(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("grants the welcome bonus to the account that opened", func(t *testing.T) {
			handler, r := newReaction()
			accountID := uuid.New()

			err := r.Handle(context.Background(), openedEvent("event-1", accountID))

			require.NoError(t, err)
			require.Equal(t, accountCommand.GrantWelcomeBonus{AccountID: accountID, Amount: welcomebonus.Amount}, handler.TriggeredWithCommand().Payload)
		})

		t.Run("sends the same command ID when the same event arrives again", func(t *testing.T) {
			handler, r := newReaction()
			accountID := uuid.New()

			require.NoError(t, r.Handle(context.Background(), openedEvent("event-1", accountID)))
			first := handler.TriggeredWithCommand().ID
			require.NoError(t, r.Handle(context.Background(), openedEvent("event-1", accountID)))

			require.Equal(t, first, handler.TriggeredWithCommand().ID)
		})
	})
}

func newReaction() (*fake.Handler[accountCommand.GrantWelcomeBonus], *welcomebonus.Reaction) {
	handler := fake.NewHandler[accountCommand.GrantWelcomeBonus]()
	bus := command.NewBus()
	bus.Register[accountCommand.GrantWelcomeBonus](handler)
	return handler, welcomebonus.NewReaction(bus)
}

func openedEvent(id string, accountID uuid.UUID) *domain.Event {
	return &domain.Event{
		ID:      id,
		Payload: &event.AccountOpened{AccountID: accountID, Owner: "alice"},
	}
}
