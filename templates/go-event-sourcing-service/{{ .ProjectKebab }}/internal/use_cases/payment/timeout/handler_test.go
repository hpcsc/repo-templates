//go:build unit

package timeout_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/timeout"
	"github.com/stretchr/testify/require"
)

var (
	now      = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	timeouts = payment.Timeouts{Checks: time.Minute, Submission: 2 * time.Minute, Settlement: time.Hour}
)

func TestHandlers(t *testing.T) {
	t.Run("checks", func(t *testing.T) {
		t.Run("saves the retry with the deadline after the timeout of the checks", func(t *testing.T) {
			bus, s := newBus()
			paymentID := uuid.New()
			given(t, s, paymentID, requested(paymentID), domain.NewEvent(&paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: now}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.TimeOutChecks{PaymentID: paymentID, Attempt: 1}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentChecksRetried{PaymentID: paymentID, Attempt: 2, ChecksDeadline: now.Add(time.Minute)}}, payloadsOf(t, s, paymentID)[2:])
		})
	})

	t.Run("submission", func(t *testing.T) {
		t.Run("saves the retry with the deadline after the timeout of the submission", func(t *testing.T) {
			bus, s := newBus()
			paymentID := uuid.New()
			given(t, s, paymentID, requested(paymentID), domain.NewEvent(&paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: now}), domain.NewEvent(&paymentEvent.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: now}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.TimeOutSubmission{PaymentID: paymentID, Attempt: 1}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentSubmissionRetried{PaymentID: paymentID, Attempt: 2, SubmissionDeadline: now.Add(2 * time.Minute)}}, payloadsOf(t, s, paymentID)[3:])
		})
	})

	t.Run("settlement", func(t *testing.T) {
		t.Run("saves the status query with the deadline after the timeout of the settlement", func(t *testing.T) {
			bus, s := newBus()
			paymentID := uuid.New()
			given(t, s, paymentID, requested(paymentID), domain.NewEvent(&paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: now}), domain.NewEvent(&paymentEvent.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: now}), domain.NewEvent(&paymentEvent.PaymentSubmitted{PaymentID: paymentID, NetworkReference: "N-8812", SettlementDeadline: now}))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.TimeOutSettlement{PaymentID: paymentID, Attempt: 1}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentStatusQueried{PaymentID: paymentID, Attempt: 2, SettlementDeadline: now.Add(time.Hour)}}, payloadsOf(t, s, paymentID)[4:])
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	timeout.Register(bus, payment.NewRepository(s), timeouts, func() time.Time { return now })
	return bus, s
}

func given(t *testing.T, s event.Stream, paymentID uuid.UUID, history ...domain.Event) {
	require.NoError(t, s.Save(context.Background(), payment.StreamID(paymentID), history, 0))
}

func requested(paymentID uuid.UUID) domain.Event {
	return domain.NewEvent(&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300})
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
