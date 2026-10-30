//go:build unit

package submission_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/common/test"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command/fake"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/stream"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/outside"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/submission"
	"github.com/stretchr/testify/require"
)

func TestGateway(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("sends the payment to the network with a key and the webhook", func(t *testing.T) {
			var key string
			var body submission.SubmitRequest
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				key = r.Header.Get("Idempotency-Key")
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"reference": "N-8812"}`))
			}))
			defer network.Close()
			paymentID := uuid.New()
			gateway, _ := newGateway(t, network.URL, paymentID, test.DiscardLogger())

			require.NoError(t, gateway.Handle(context.Background(), cleared(paymentID)))

			require.Equal(t, outside.KeyFor(paymentID, "submission"), key)
			require.Equal(t, submission.SubmitRequest{PaymentID: paymentID, Amount: 300, CreditorIBAN: "DE89370400440532013000", WebhookURL: "http://api/webhooks/payment-network"}, body)
		})

		t.Run("sends a retried payment with the key of the first attempt", func(t *testing.T) {
			var keys []string
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				keys = append(keys, r.Header.Get("Idempotency-Key"))
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"reference": "N-8812"}`))
			}))
			defer network.Close()
			paymentID := uuid.New()
			gateway, _ := newGateway(t, network.URL, paymentID, test.DiscardLogger())

			require.NoError(t, gateway.Handle(context.Background(), cleared(paymentID)))
			require.NoError(t, gateway.Handle(context.Background(), &domain.Event{ID: "event-2", Payload: &paymentEvent.PaymentSubmissionRetried{PaymentID: paymentID, Attempt: 2, SubmissionDeadline: time.Now()}}))

			require.Equal(t, []string{outside.KeyFor(paymentID, "submission"), outside.KeyFor(paymentID, "submission")}, keys)
		})

		t.Run("records the reference that the network gives", func(t *testing.T) {
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"reference": "N-8812"}`))
			}))
			defer network.Close()
			paymentID := uuid.New()
			gateway, submissions := newGateway(t, network.URL, paymentID, test.DiscardLogger())

			err := gateway.Handle(context.Background(), cleared(paymentID))

			require.NoError(t, err)
			require.Equal(t, paymentCommand.RecordSubmission{PaymentID: paymentID, NetworkReference: "N-8812"}, submissions.TriggeredWithCommand().Payload)
		})

		t.Run("logs a refused payment, records nothing, and returns no error, so the consumer goes on", func(t *testing.T) {
			network := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer network.Close()
			paymentID := uuid.New()
			logs := test.NewFakeLogHandler()
			gateway, submissions := newGateway(t, network.URL, paymentID, slog.New(logs))

			err := gateway.Handle(context.Background(), cleared(paymentID))

			require.NoError(t, err)
			require.Zero(t, submissions.Calls())
			require.Contains(t, logs.Record.Message, "failed to send payment "+paymentID.String())
		})
	})
}

func newGateway(t *testing.T, networkURL string, paymentID uuid.UUID, logger *slog.Logger) (*submission.Gateway, *fake.Handler[paymentCommand.RecordSubmission]) {
	s := stream.NewMemory()
	require.NoError(t, s.Save(context.Background(), payment.StreamID(paymentID), []domain.Event{
		domain.NewEvent(&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}),
	}, 0))
	submissions := fake.NewHandler[paymentCommand.RecordSubmission]()
	bus := command.NewBus()
	bus.Register[paymentCommand.RecordSubmission](submissions)
	cfg := outside.Config{PaymentNetworkURL: networkURL, WebhookURL: "http://api/webhooks"}
	return submission.NewGateway(bus, payment.NewRepository(s), outside.NewClient(outside.WithRetries(1, time.Millisecond)), cfg, logger), submissions
}

func cleared(paymentID uuid.UUID) *domain.Event {
	return &domain.Event{ID: "event-1", Payload: &paymentEvent.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: time.Now()}}
}
