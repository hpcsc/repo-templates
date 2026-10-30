//go:build unit

package webhook_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	apiTest "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/webhook"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("fraud checks", func(t *testing.T) {
		t.Run("records the result of the fraud check", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.RecordFraudResult]()
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordFraudResult](handler)
			paymentID := uuid.New()

			w := send(bus, "/webhooks/fraud-checks", fmt.Sprintf(`{"paymentId": %q, "passed": false, "reason": "amount above the limit"}`, paymentID), "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, paymentCommand.RecordFraudResult{PaymentID: paymentID, Passed: false, Reason: "amount above the limit"}, handler.TriggeredWithCommand().Payload)
		})

		t.Run("gives a repeated webhook the same command ID", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.RecordFraudResult]()
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordFraudResult](handler)
			body := fmt.Sprintf(`{"paymentId": %q, "passed": true}`, uuid.New())

			send(bus, "/webhooks/fraud-checks", body, "delivery-1")
			first := handler.TriggeredWithCommand().ID
			send(bus, "/webhooks/fraud-checks", body, "delivery-1")

			require.Equal(t, first, handler.TriggeredWithCommand().ID)
		})

		t.Run("returns not found for a payment that is not requested", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordFraudResult](fake.NewHandler[paymentCommand.RecordFraudResult]().WithError(payment.ErrNotRequested))

			w := send(bus, "/webhooks/fraud-checks", fmt.Sprintf(`{"paymentId": %q, "passed": true}`, uuid.New()), "")

			require.Equal(t, http.StatusNotFound, w.Code)
			apiTest.RequireErrorResponse(t, w, "payment not found")
		})

		t.Run("rejects a payment ID that is not a UUID", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordFraudResult](fake.NewHandler[paymentCommand.RecordFraudResult]())

			w := send(bus, "/webhooks/fraud-checks", `{"paymentId": "not-a-uuid", "passed": true}`, "")

			require.Equal(t, http.StatusBadRequest, w.Code)
			apiTest.RequireErrorResponse(t, w, "must be a payment ID")
		})
	})

	t.Run("sanctions screenings", func(t *testing.T) {
		t.Run("records the result of the screening", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.RecordScreeningResult]()
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordScreeningResult](handler)
			paymentID := uuid.New()

			w := send(bus, "/webhooks/sanctions-screenings", fmt.Sprintf(`{"paymentId": %q, "passed": true}`, paymentID), "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, paymentCommand.RecordScreeningResult{PaymentID: paymentID, Passed: true}, handler.TriggeredWithCommand().Payload)
		})
	})

	t.Run("payment network", func(t *testing.T) {
		t.Run("records the settlement of a settled payment", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.RecordSettlement]()
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordSettlement](handler)
			paymentID := uuid.New()

			w := send(bus, "/webhooks/payment-network", fmt.Sprintf(`{"paymentId": %q, "status": "settled"}`, paymentID), "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, paymentCommand.RecordSettlement{PaymentID: paymentID}, handler.TriggeredWithCommand().Payload)
		})

		t.Run("records the return with the reason of the other bank", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.RecordReturn]()
			bus := command.NewBus()
			bus.Register[paymentCommand.RecordReturn](handler)
			paymentID := uuid.New()

			w := send(bus, "/webhooks/payment-network", fmt.Sprintf(`{"paymentId": %q, "status": "returned", "reason": "account closed at the other bank"}`, paymentID), "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, paymentCommand.RecordReturn{PaymentID: paymentID, Reason: "account closed at the other bank"}, handler.TriggeredWithCommand().Payload)
		})

		t.Run("rejects a status that is not settled or returned", func(t *testing.T) {
			w := send(command.NewBus(), "/webhooks/payment-network", fmt.Sprintf(`{"paymentId": %q, "status": "pending"}`, uuid.New()), "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}

func send(bus *command.Bus, path string, body string, key string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	apiTest.NewRouterWithRoutable(webhook.NewHandler(bus)).ServeHTTP(w, req)
	return w
}
