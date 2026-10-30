package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/idempotency"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/response"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/api/route"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/summary"
	"github.com/unrolled/render"
)

type SummaryReader interface {
	ByID(ctx context.Context, id string) (*summary.Summary, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, summaries SummaryReader) route.Routable {
	return &handler{
		renderer:  render.New(),
		bus:       bus,
		summaries: summaries,
	}
}

type handler struct {
	renderer  *render.Render
	bus       *command.Bus
	summaries SummaryReader
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/payments", h.request),
		route.Protected("GET", "/payments/{id}", h.get),
	}
}

func (h *handler) request(w http.ResponseWriter, req *http.Request) {
	var body RequestPayment
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return
	}

	v := validate.Struct(body)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return
	}

	accountID, err := uuid.Parse(body.AccountID)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("accountId must be an account ID"))
		return
	}

	paymentID := idempotency.IDFor(req, "payment")
	err = h.bus.Dispatch(req.Context(), domain.Command[paymentCommand.Request]{
		ID: idempotency.IDFor(req, "command"),
		Payload: paymentCommand.Request{
			PaymentID:    paymentID,
			AccountID:    accountID,
			CreditorIBAN: body.CreditorIBAN,
			Amount:       body.Amount,
		},
	})
	if errors.Is(err, event.ErrConcurrencyConflict) {
		_ = h.renderer.JSON(w, http.StatusConflict, response.Fail(fmt.Sprintf("%v, retry the request", err)))
		return
	}
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": paymentID.String()}))
}

func (h *handler) get(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if _, err := uuid.Parse(id); err != nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("payment not found"))
		return
	}

	s, err := h.summaries.ByID(req.Context(), id)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if s == nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("payment not found"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(s))
}
