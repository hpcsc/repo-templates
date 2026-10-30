//go:build unit

package payment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/stream"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	t.Run("requested", func(t *testing.T) {
		t.Run("returns the request of the payment after later events", func(t *testing.T) {
			s := stream.NewMemory()
			id, accountID := uuid.New(), uuid.New()
			require.NoError(t, s.Save(context.Background(), payment.StreamID(id), []domain.Event{
				domain.NewEvent(&event.PaymentRequested{PaymentID: id, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}),
				domain.NewEvent(&event.PaymentFunded{PaymentID: id, ChecksDeadline: deadline}),
			}, 0))

			requested, err := payment.NewRepository(s).Requested(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, &event.PaymentRequested{PaymentID: id, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}, requested)
		})

		t.Run("returns the error for a payment that is not requested", func(t *testing.T) {
			_, err := payment.NewRepository(stream.NewMemory()).Requested(context.Background(), uuid.New())

			require.ErrorIs(t, err, payment.ErrNotRequested)
		})
	})

	t.Run("submitted", func(t *testing.T) {
		t.Run("returns the submission with the reference of the network", func(t *testing.T) {
			s := stream.NewMemory()
			id := uuid.New()
			require.NoError(t, s.Save(context.Background(), payment.StreamID(id), []domain.Event{
				domain.NewEvent(&event.PaymentRequested{PaymentID: id, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}),
				domain.NewEvent(&event.PaymentSubmitted{PaymentID: id, NetworkReference: "N-8812", SettlementDeadline: deadline}),
			}, 0))

			submitted, err := payment.NewRepository(s).Submitted(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, &event.PaymentSubmitted{PaymentID: id, NetworkReference: "N-8812", SettlementDeadline: deadline}, submitted)
		})

		t.Run("returns the error for a payment that is not submitted", func(t *testing.T) {
			s := stream.NewMemory()
			id := uuid.New()
			require.NoError(t, s.Save(context.Background(), payment.StreamID(id), []domain.Event{
				domain.NewEvent(&event.PaymentRequested{PaymentID: id, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}),
			}, 0))

			_, err := payment.NewRepository(s).Submitted(context.Background(), id)

			require.ErrorIs(t, err, payment.ErrNotSubmitted)
		})
	})
}
