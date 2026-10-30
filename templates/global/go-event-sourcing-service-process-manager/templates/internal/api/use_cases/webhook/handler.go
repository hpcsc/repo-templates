package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/idempotency"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/response"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/route"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	"github.com/unrolled/render"
)

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/webhooks/fraud-checks", h.fraudCheck),
		route.Protected("POST", "/webhooks/sanctions-screenings", h.sanctionsScreening),
		route.Protected("POST", "/webhooks/payment-network", h.paymentNetwork),
	}
}

func (h *handler) fraudCheck(w http.ResponseWriter, req *http.Request) {
	body, paymentID, ok := decode[CheckResult](h, w, req)
	if !ok {
		return
	}
	dispatch(h, w, req, paymentCommand.RecordFraudResult{PaymentID: paymentID, Passed: body.Passed, Reason: body.Reason})
}

func (h *handler) sanctionsScreening(w http.ResponseWriter, req *http.Request) {
	body, paymentID, ok := decode[CheckResult](h, w, req)
	if !ok {
		return
	}
	dispatch(h, w, req, paymentCommand.RecordScreeningResult{PaymentID: paymentID, Passed: body.Passed, Reason: body.Reason})
}

func (h *handler) paymentNetwork(w http.ResponseWriter, req *http.Request) {
	body, paymentID, ok := decode[NetworkStatus](h, w, req)
	if !ok {
		return
	}
	if body.Status == "settled" {
		dispatch(h, w, req, paymentCommand.RecordSettlement{PaymentID: paymentID})
		return
	}
	dispatch(h, w, req, paymentCommand.RecordReturn{PaymentID: paymentID, Reason: body.Reason})
}

type withPaymentID interface {
	paymentID() string
}

func decode[T withPaymentID](h *handler, w http.ResponseWriter, req *http.Request) (*T, uuid.UUID, bool) {
	var body T
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return nil, uuid.Nil, false
	}

	v := validate.Struct(body)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return nil, uuid.Nil, false
	}

	paymentID, err := uuid.Parse(body.paymentID())
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("paymentId must be a payment ID"))
		return nil, uuid.Nil, false
	}
	return &body, paymentID, true
}

func dispatch[P any](h *handler, w http.ResponseWriter, req *http.Request, payload P) {
	err := h.bus.Dispatch(req.Context(), domain.Command[P]{
		ID:      idempotency.IDFor(req, "command"),
		Payload: payload,
	})
	switch {
	case errors.Is(err, payment.ErrNotRequested):
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("payment not found"))
	case errors.Is(err, event.ErrConcurrencyConflict):
		_ = h.renderer.JSON(w, http.StatusConflict, response.Fail(fmt.Sprintf("%v, retry the request", err)))
	case err != nil:
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
	default:
		_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
	}
}
