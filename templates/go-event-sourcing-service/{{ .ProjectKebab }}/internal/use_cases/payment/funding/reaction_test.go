//go:build unit

package funding_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/funding"
	"github.com/stretchr/testify/require"
)

func TestReaction(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("asks the account to hold the amount of a requested payment", func(t *testing.T) {
			r, holds, _ := newReaction()
			paymentID, accountID := uuid.New(), uuid.New()

			err := r.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: &paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}})

			require.NoError(t, err)
			require.Equal(t, accountCommand.PlaceHold{AccountID: accountID, PaymentID: paymentID, Amount: 300}, holds.TriggeredWithCommand().Payload)
		})

		t.Run("records the funding of the payment when the account holds the amount", func(t *testing.T) {
			r, _, fundings := newReaction()
			paymentID := uuid.New()

			err := r.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: &accountEvent.HoldPlaced{AccountID: uuid.New(), PaymentID: paymentID, Amount: 300}})

			require.NoError(t, err)
			require.Equal(t, paymentCommand.RecordFunding{PaymentID: paymentID, Held: true}, fundings.TriggeredWithCommand().Payload)
		})

		t.Run("records the funding with the reason of the account when the account rejects the hold", func(t *testing.T) {
			r, _, fundings := newReaction()
			paymentID := uuid.New()

			err := r.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: &accountEvent.HoldRejected{AccountID: uuid.New(), PaymentID: paymentID, Amount: 300, Reason: "account is closed"}})

			require.NoError(t, err)
			require.Equal(t, paymentCommand.RecordFunding{PaymentID: paymentID, Held: false, Reason: "account is closed"}, fundings.TriggeredWithCommand().Payload)
		})

		t.Run("sends the same command ID when the same event arrives again", func(t *testing.T) {
			r, _, fundings := newReaction()
			placed := &domain.Event{ID: "event-1", Payload: &accountEvent.HoldPlaced{AccountID: uuid.New(), PaymentID: uuid.New(), Amount: 300}}

			require.NoError(t, r.Handle(context.Background(), placed))
			first := fundings.TriggeredWithCommand().ID
			require.NoError(t, r.Handle(context.Background(), placed))

			require.Equal(t, first, fundings.TriggeredWithCommand().ID)
		})
	})
}

func newReaction() (*funding.Reaction, *fake.Handler[accountCommand.PlaceHold], *fake.Handler[paymentCommand.RecordFunding]) {
	holds := fake.NewHandler[accountCommand.PlaceHold]()
	fundings := fake.NewHandler[paymentCommand.RecordFunding]()
	bus := command.NewBus()
	bus.Register[accountCommand.PlaceHold](holds)
	bus.Register[paymentCommand.RecordFunding](fundings)
	return funding.NewReaction(bus), holds, fundings
}
