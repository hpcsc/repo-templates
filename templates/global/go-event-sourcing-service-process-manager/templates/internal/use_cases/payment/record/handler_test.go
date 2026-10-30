//go:build unit

package record_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/stream"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/record"
	"github.com/stretchr/testify/require"
)

var (
	now      = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	timeouts = payment.Timeouts{Checks: time.Minute, Submission: 2 * time.Minute, Settlement: time.Hour}
)

func TestHandlers(t *testing.T) {
	t.Run("funding", func(t *testing.T) {
		t.Run("saves the funding with the deadline of the checks after the timeout of the checks", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordFunding{PaymentID: paymentID, Held: true}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: now.Add(time.Minute)}}, payloadsOf(t, s, paymentID)[1:])
		})
	})

	t.Run("fraud result", func(t *testing.T) {
		t.Run("saves the clearance with the deadline of the submission after the timeout of the submission", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID), funded(paymentID), domain.NewEvent(&paymentEvent.SanctionsScreeningPassed{PaymentID: paymentID}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordFraudResult{PaymentID: paymentID, Passed: true}))

			require.NoError(t, err)
			require.Equal(t, []any{
				&paymentEvent.FraudCheckPassed{PaymentID: paymentID},
				&paymentEvent.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: now.Add(2 * time.Minute)},
			}, payloadsOf(t, s, paymentID)[3:])
		})
	})

	t.Run("screening result", func(t *testing.T) {
		t.Run("saves the refusal when the screening fails", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID), funded(paymentID))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordScreeningResult{PaymentID: paymentID, Passed: false, Reason: "name on the sanctions list"}))

			require.NoError(t, err)
			require.Equal(t, []any{
				&paymentEvent.SanctionsScreeningFailed{PaymentID: paymentID, Reason: "name on the sanctions list"},
				&paymentEvent.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "name on the sanctions list"},
			}, payloadsOf(t, s, paymentID)[2:])
		})
	})

	t.Run("submission", func(t *testing.T) {
		t.Run("saves the submission with the deadline of the settlement after the timeout of the settlement", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID), funded(paymentID), cleared(paymentID))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordSubmission{PaymentID: paymentID, NetworkReference: "N-8812"}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentSubmitted{PaymentID: paymentID, NetworkReference: "N-8812", SettlementDeadline: now.Add(time.Hour)}}, payloadsOf(t, s, paymentID)[3:])
		})
	})

	t.Run("settlement", func(t *testing.T) {
		t.Run("saves the settlement", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID), funded(paymentID), cleared(paymentID))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordSettlement{PaymentID: paymentID}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentSettled{PaymentID: paymentID, AccountID: accountID}}, payloadsOf(t, s, paymentID)[3:])
		})

		t.Run("returns the error for a payment that is not requested", func(t *testing.T) {
			bus, _ := newBus()

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordSettlement{PaymentID: uuid.New()}))

			require.ErrorIs(t, err, payment.ErrNotRequested)
		})
	})

	t.Run("return", func(t *testing.T) {
		t.Run("saves the return with the reason of the other bank", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			given(t, s, paymentID, requested(paymentID, accountID), funded(paymentID), cleared(paymentID))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.RecordReturn{PaymentID: paymentID, Reason: "account closed at the other bank"}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentReturned{PaymentID: paymentID, AccountID: accountID, Reason: "account closed at the other bank"}}, payloadsOf(t, s, paymentID)[3:])
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	record.Register(bus, payment.NewRepository(s), timeouts, func() time.Time { return now })
	return bus, s
}

func given(t *testing.T, s event.Stream, paymentID uuid.UUID, history ...domain.Event) {
	require.NoError(t, s.Save(context.Background(), payment.StreamID(paymentID), history, 0))
}

func requested(paymentID, accountID uuid.UUID) domain.Event {
	return domain.NewEvent(&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300})
}

func funded(paymentID uuid.UUID) domain.Event {
	return domain.NewEvent(&paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: now})
}

func cleared(paymentID uuid.UUID) domain.Event {
	return domain.NewEvent(&paymentEvent.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: now})
}

func payloadsOf(t *testing.T, s event.Stream, paymentID uuid.UUID) []any {
	events, err := s.EventsForStream(context.Background(), payment.StreamID(paymentID))
	require.NoError(t, err)
	payloads := []any{}
	for _, e := range events {
		payloads = append(payloads, e.Payload)
	}
	return payloads
}
