//go:build unit

package status_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/outside"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/status"
	"github.com/stretchr/testify/require"
)

func TestGateway(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("asks for the status of the payment by the reference of the network", func(t *testing.T) {
			var path string
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				_, _ = w.Write([]byte(`{"status": "pending"}`))
			}))
			defer network.Close()
			paymentID := uuid.New()
			sys := newSystem(t, network.URL, paymentID, test.DiscardLogger())

			require.NoError(t, sys.gateway.Handle(context.Background(), queried(paymentID)))

			require.Equal(t, "/payments/N-8812", path)
		})

		t.Run("records the settlement when the network settled the payment", func(t *testing.T) {
			network := answering(`{"status": "settled"}`)
			defer network.Close()
			paymentID := uuid.New()
			sys := newSystem(t, network.URL, paymentID, test.DiscardLogger())

			err := sys.gateway.Handle(context.Background(), queried(paymentID))

			require.NoError(t, err)
			require.Equal(t, paymentCommand.RecordSettlement{PaymentID: paymentID}, sys.settlements.TriggeredWithCommand().Payload)
		})

		t.Run("records the return with the reason of the other bank", func(t *testing.T) {
			network := answering(`{"status": "returned", "reason": "account closed at the other bank"}`)
			defer network.Close()
			paymentID := uuid.New()
			sys := newSystem(t, network.URL, paymentID, test.DiscardLogger())

			err := sys.gateway.Handle(context.Background(), queried(paymentID))

			require.NoError(t, err)
			require.Equal(t, paymentCommand.RecordReturn{PaymentID: paymentID, Reason: "account closed at the other bank"}, sys.returns.TriggeredWithCommand().Payload)
		})

		t.Run("records nothing while the network has no answer", func(t *testing.T) {
			network := answering(`{"status": "pending"}`)
			defer network.Close()
			paymentID := uuid.New()
			sys := newSystem(t, network.URL, paymentID, test.DiscardLogger())

			err := sys.gateway.Handle(context.Background(), queried(paymentID))

			require.NoError(t, err)
			require.Zero(t, sys.settlements.Calls()+sys.returns.Calls())
		})

		t.Run("logs a failed request and returns no error, so the consumer goes on", func(t *testing.T) {
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer network.Close()
			paymentID := uuid.New()
			logs := test.NewFakeLogHandler()
			sys := newSystem(t, network.URL, paymentID, slog.New(logs))

			err := sys.gateway.Handle(context.Background(), queried(paymentID))

			require.NoError(t, err)
			require.Contains(t, logs.Record.Message, "failed to ask for the status of payment "+paymentID.String())
		})
	})
}

type system struct {
	gateway     *status.Gateway
	settlements *fake.Handler[paymentCommand.RecordSettlement]
	returns     *fake.Handler[paymentCommand.RecordReturn]
}

func newSystem(t *testing.T, networkURL string, paymentID uuid.UUID, logger *slog.Logger) system {
	s := stream.NewMemory()
	require.NoError(t, s.Save(context.Background(), payment.StreamID(paymentID), []domain.Event{
		domain.NewEvent(&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}),
		domain.NewEvent(&paymentEvent.PaymentSubmitted{PaymentID: paymentID, NetworkReference: "N-8812", SettlementDeadline: time.Now()}),
	}, 0))
	settlements := fake.NewHandler[paymentCommand.RecordSettlement]()
	returns := fake.NewHandler[paymentCommand.RecordReturn]()
	bus := command.NewBus()
	bus.Register[paymentCommand.RecordSettlement](settlements)
	bus.Register[paymentCommand.RecordReturn](returns)
	cfg := outside.Config{PaymentNetworkURL: networkURL}
	gateway := status.NewGateway(bus, payment.NewRepository(s), outside.NewClient(outside.WithRetries(1, time.Millisecond)), cfg, logger)
	return system{gateway: gateway, settlements: settlements, returns: returns}
}

func answering(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

func queried(paymentID uuid.UUID) *domain.Event {
	return &domain.Event{ID: "event-1", Payload: &paymentEvent.PaymentStatusQueried{PaymentID: paymentID, Attempt: 2, SettlementDeadline: time.Now()}}
}
