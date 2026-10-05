package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/response"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/balance"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	"github.com/unrolled/render"
)

type BalanceReader interface {
	ByID(ctx context.Context, id string) (*balance.Balance, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, balances BalanceReader) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
		balances: balances,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
	balances BalanceReader
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/accounts", h.open),
		route.Protected("POST", "/accounts/{id}/deposits", h.deposit),
		route.Protected("POST", "/accounts/{id}/withdrawals", h.withdraw),
		route.Protected("POST", "/accounts/{id}/close", h.close),
		route.Protected("GET", "/accounts/{id}", h.get),
	}
}

func (h *handler) open(w http.ResponseWriter, req *http.Request) {
	var body OpenRequest
	if !h.decode(w, req, &body) {
		return
	}

	accountID := uuid.New()
	if !h.dispatch(w, req, accountCommand.Open{AccountID: accountID, Owner: body.Owner}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": accountID.String()}))
}

func (h *handler) deposit(w http.ResponseWriter, req *http.Request) {
	accountID, body, ok := h.amountRequest(w, req)
	if !ok {
		return
	}

	if h.dispatch(w, req, accountCommand.Deposit{AccountID: accountID, Amount: body.Amount}) {
		_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
	}
}

func (h *handler) withdraw(w http.ResponseWriter, req *http.Request) {
	accountID, body, ok := h.amountRequest(w, req)
	if !ok {
		return
	}

	if h.dispatch(w, req, accountCommand.Withdraw{AccountID: accountID, Amount: body.Amount}) {
		_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
	}
}

func (h *handler) close(w http.ResponseWriter, req *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("account ID is not a UUID"))
		return
	}

	if h.dispatch(w, req, accountCommand.Close{AccountID: accountID}) {
		_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
	}
}

func (h *handler) get(w http.ResponseWriter, req *http.Request) {
	b, err := h.balances.ByID(req.Context(), chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if b == nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("account not found"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(b))
}

func (h *handler) amountRequest(w http.ResponseWriter, req *http.Request) (uuid.UUID, AmountRequest, bool) {
	var body AmountRequest
	accountID, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("account ID is not a UUID"))
		return uuid.Nil, body, false
	}

	return accountID, body, h.decode(w, req, &body)
}

func (h *handler) decode(w http.ResponseWriter, req *http.Request, body any) bool {
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return false
	}

	v := validate.Struct(body)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return false
	}
	return true
}

func (h *handler) dispatch[P any](w http.ResponseWriter, req *http.Request, payload P) bool {
	err := h.bus.Dispatch(req.Context(), domain.NewCommand(payload))
	if errors.Is(err, event.ErrConcurrencyConflict) {
		_ = h.renderer.JSON(w, http.StatusConflict, response.Fail(fmt.Sprintf("%v, retry the request", err)))
		return false
	}
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return false
	}
	return true
}
