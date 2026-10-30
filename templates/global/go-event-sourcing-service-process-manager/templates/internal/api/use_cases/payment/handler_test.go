//go:build unit

package payment_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/response"
	apiTest "github.com/hpcsc/go-event-sourcing-service-project/internal/api/test"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/use_cases/payment"
	commonTest "github.com/hpcsc/go-event-sourcing-service-project/internal/common/test"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command/fake"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/summary"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		t.Run("requests the payment and returns the new payment ID", func(t *testing.T) {
			handler := fake.NewHandler[paymentCommand.Request]()
			bus := command.NewBus()
			bus.Register[paymentCommand.Request](handler)
			accountID := uuid.New()

			w := send(newRouter(bus, nil), "POST", "/payments", fmt.Sprintf(`{"accountId": %q, "creditorIban": "DE89370400440532013000", "amount": 300}`, accountID))

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, paymentCommand.Request{PaymentID: payload.PaymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}, payload)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.Equal(t, payload.PaymentID.String(), body.Data.(map[string]any)["id"])
		})

		t.Run("rejects an account ID that is not a UUID", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[paymentCommand.Request](fake.NewHandler[paymentCommand.Request]())

			w := send(newRouter(bus, nil), "POST", "/payments", `{"accountId": "not-a-uuid", "creditorIban": "DE89370400440532013000", "amount": 300}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
			apiTest.RequireErrorResponse(t, w, "must be an account ID")
		})

		t.Run("returns the refusal of the payment", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[paymentCommand.Request](fake.NewHandler[paymentCommand.Request]().WithError(fmt.Errorf("account does not exist")))

			w := send(newRouter(bus, nil), "POST", "/payments", fmt.Sprintf(`{"accountId": %q, "creditorIban": "DE89370400440532013000", "amount": 300}`, uuid.New()))

			require.Equal(t, http.StatusBadRequest, w.Code)
			apiTest.RequireErrorResponse(t, w, "account does not exist")
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Run("returns the summary of the payment", func(t *testing.T) {
			id := uuid.NewString()

			w := send(newRouter(command.NewBus(), &summary.Summary{ID: id, Status: "escalated", Reason: "the payment network did not settle the payment"}), "GET", "/payments/"+id, "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"status":"escalated"`)
			require.Contains(t, w.Body.String(), `"reason":"the payment network did not settle the payment"`)
		})

		t.Run("returns not found when the read model has no row for the payment", func(t *testing.T) {
			w := send(newRouter(command.NewBus(), nil), "GET", "/payments/"+uuid.NewString(), "")

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("returns not found for an ID that is not a UUID", func(t *testing.T) {
			w := send(newRouter(command.NewBus(), &summary.Summary{ID: "not-a-uuid"}), "GET", "/payments/not-a-uuid", "")

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}

type fakeSummaries struct {
	summary *summary.Summary
}

func (f *fakeSummaries) ByID(context.Context, string) (*summary.Summary, error) {
	return f.summary, nil
}

func newRouter(bus *command.Bus, s *summary.Summary) http.Handler {
	return apiTest.NewRouterWithRoutable(payment.NewHandler(bus, &fakeSummaries{summary: s}))
}

func send(router http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}
