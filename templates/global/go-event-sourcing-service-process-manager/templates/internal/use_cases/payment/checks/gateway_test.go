//go:build unit

package checks_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/common/test"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/stream"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/checks"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/outside"
	"github.com/stretchr/testify/require"
)

func TestGateway(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("asks the fraud service and the sanctions service, each with its own key and webhook", func(t *testing.T) {
			services := newServices(http.StatusAccepted)
			defer services.Close()
			paymentID := uuid.New()
			gateway := newGateway(t, services.URL, paymentID, test.DiscardLogger())

			err := gateway.Handle(context.Background(), funded(paymentID))

			require.NoError(t, err)
			require.ElementsMatch(t, []request{
				{path: "/fraud/checks", key: outside.KeyFor(paymentID, "fraud-check/1"), body: checks.CheckRequest{PaymentID: paymentID, Amount: 300, CreditorIBAN: "DE89370400440532013000", WebhookURL: "http://api/webhooks/fraud-checks"}},
				{path: "/sanctions/screenings", key: outside.KeyFor(paymentID, "sanctions-screening/1"), body: checks.CheckRequest{PaymentID: paymentID, Amount: 300, CreditorIBAN: "DE89370400440532013000", WebhookURL: "http://api/webhooks/sanctions-screenings"}},
			}, services.requests())
		})

		t.Run("asks again with a new key for each attempt", func(t *testing.T) {
			services := newServices(http.StatusAccepted)
			defer services.Close()
			paymentID := uuid.New()
			gateway := newGateway(t, services.URL, paymentID, test.DiscardLogger())

			err := gateway.Handle(context.Background(), &domain.Event{ID: "event-2", Payload: &paymentEvent.PaymentChecksRetried{PaymentID: paymentID, Attempt: 2, ChecksDeadline: time.Now()}})

			require.NoError(t, err)
			require.ElementsMatch(t, []string{outside.KeyFor(paymentID, "fraud-check/2"), outside.KeyFor(paymentID, "sanctions-screening/2")}, services.keys())
		})

		t.Run("logs a refused request and returns no error, so the consumer goes on", func(t *testing.T) {
			services := newServices(http.StatusBadRequest)
			defer services.Close()
			paymentID := uuid.New()
			logs := test.NewFakeLogHandler()

			err := newGateway(t, services.URL, paymentID, slog.New(logs)).Handle(context.Background(), funded(paymentID))

			require.NoError(t, err)
			require.Contains(t, logs.Record.Message, "failed to ask for the checks of payment "+paymentID.String())
		})
	})
}

func newGateway(t *testing.T, servicesURL string, paymentID uuid.UUID, logger *slog.Logger) *checks.Gateway {
	s := stream.NewMemory()
	require.NoError(t, s.Save(context.Background(), payment.StreamID(paymentID), []domain.Event{
		domain.NewEvent(&paymentEvent.PaymentRequested{PaymentID: paymentID, AccountID: uuid.New(), CreditorIBAN: "DE89370400440532013000", Amount: 300}),
	}, 0))
	cfg := outside.Config{
		FraudServiceURL:     servicesURL + "/fraud",
		SanctionsServiceURL: servicesURL + "/sanctions",
		WebhookURL:          "http://api/webhooks",
	}
	return checks.NewGateway(payment.NewRepository(s), outside.NewClient(outside.WithRetries(1, time.Millisecond)), cfg, logger)
}

func funded(paymentID uuid.UUID) *domain.Event {
	return &domain.Event{ID: "event-1", Payload: &paymentEvent.PaymentFunded{PaymentID: paymentID, ChecksDeadline: time.Now()}}
}

type request struct {
	path string
	key  string
	body checks.CheckRequest
}

type services struct {
	*httptest.Server
	mutex    sync.Mutex
	received []request
}

func newServices(status int) *services {
	s := &services{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body checks.CheckRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mutex.Lock()
		s.received = append(s.received, request{path: r.URL.Path, key: r.Header.Get("Idempotency-Key"), body: body})
		s.mutex.Unlock()
		w.WriteHeader(status)
	}))
	return s
}

func (s *services) requests() []request {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]request(nil), s.received...)
}

func (s *services) keys() []string {
	var keys []string
	for _, r := range s.requests() {
		keys = append(keys, r.key)
	}
	return keys
}
