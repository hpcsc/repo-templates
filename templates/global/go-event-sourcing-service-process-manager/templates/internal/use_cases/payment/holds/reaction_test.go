//go:build unit

package holds_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command/fake"
	accountCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account/command"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/holds"
	"github.com/stretchr/testify/require"
)

func TestReaction(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("captures the hold of a settled payment", func(t *testing.T) {
			r, captures, _ := newReaction()
			paymentID, accountID := uuid.New(), uuid.New()

			err := r.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: &paymentEvent.PaymentSettled{PaymentID: paymentID, AccountID: accountID}})

			require.NoError(t, err)
			require.Equal(t, accountCommand.CaptureHold{AccountID: accountID, PaymentID: paymentID}, captures.TriggeredWithCommand().Payload)
		})

		for name, end := range map[string]func(paymentID, accountID uuid.UUID) domain.EventPayload{
			"refused": func(paymentID, accountID uuid.UUID) domain.EventPayload {
				return &paymentEvent.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "name on the sanctions list"}
			},
			"failed": func(paymentID, accountID uuid.UUID) domain.EventPayload {
				return &paymentEvent.PaymentFailed{PaymentID: paymentID, AccountID: accountID, Reason: "account is closed"}
			},
			"returned": func(paymentID, accountID uuid.UUID) domain.EventPayload {
				return &paymentEvent.PaymentReturned{PaymentID: paymentID, AccountID: accountID, Reason: "account closed at the other bank"}
			},
		} {
			t.Run("releases the hold of a "+name+" payment", func(t *testing.T) {
				r, _, releases := newReaction()
				paymentID, accountID := uuid.New(), uuid.New()

				err := r.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: end(paymentID, accountID)})

				require.NoError(t, err)
				require.Equal(t, accountCommand.ReleaseHold{AccountID: accountID, PaymentID: paymentID}, releases.TriggeredWithCommand().Payload)
			})
		}
	})
}

func newReaction() (*holds.Reaction, *fake.Handler[accountCommand.CaptureHold], *fake.Handler[accountCommand.ReleaseHold]) {
	captures := fake.NewHandler[accountCommand.CaptureHold]()
	releases := fake.NewHandler[accountCommand.ReleaseHold]()
	bus := command.NewBus()
	bus.Register[accountCommand.CaptureHold](captures)
	bus.Register[accountCommand.ReleaseHold](releases)
	return holds.NewReaction(bus), captures, releases
}
