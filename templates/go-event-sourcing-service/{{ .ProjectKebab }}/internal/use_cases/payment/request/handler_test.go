//go:build unit

package request_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/request"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		t.Run("saves the request of a payment from an open account", func(t *testing.T) {
			bus, s := newBus()
			paymentID, accountID := uuid.New(), uuid.New()
			require.NoError(t, s.Save(context.Background(), account.StreamID(accountID), []domain.Event{domain.NewEvent(&accountEvent.AccountOpened{AccountID: accountID, Owner: "alice"})}, 0))

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.Request{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}))

			require.NoError(t, err)
			require.Equal(t, []any{&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}}, payloadsOf(t, s, paymentID))
		})

		t.Run("rejects a payment from an account that does not exist, and saves nothing", func(t *testing.T) {
			bus, s := newBus()
			paymentID := uuid.New()

			err := bus.Dispatch(context.Background(), domain.NewCommand(paymentCommand.Request{PaymentID: paymentID, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}))

			require.ErrorIs(t, err, payment.ErrUnknownAccount)
			require.Empty(t, payloadsOf(t, s, paymentID))
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	request.Register(bus, payment.NewRepository(s), account.NewRepository(s))
	return bus, s
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
