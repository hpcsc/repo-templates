package transfer

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
	transferCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
	"github.com/unrolled/render"
)

type StatusReader interface {
	ByID(ctx context.Context, id string) (*process.Status, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, statuses StatusReader) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
		statuses: statuses,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
	statuses StatusReader
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/transfers", h.start),
		route.Protected("GET", "/transfers/{id}", h.get),
	}
}

func (h *handler) start(w http.ResponseWriter, req *http.Request) {
	var body StartRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return
	}

	v := validate.Struct(body)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return
	}

	from, fromErr := uuid.Parse(body.From)
	to, toErr := uuid.Parse(body.To)
	if fromErr != nil || toErr != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("from and to must be account IDs"))
		return
	}

	transferID := uuid.New()
	err := h.bus.Dispatch(req.Context(), domain.NewCommand(transferCommand.Start{
		TransferID: transferID,
		From:       from,
		To:         to,
		Amount:     body.Amount,
	}))
	if errors.Is(err, event.ErrConcurrencyConflict) {
		_ = h.renderer.JSON(w, http.StatusConflict, response.Fail(fmt.Sprintf("%v, retry the request", err)))
		return
	}
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": transferID.String()}))
}

func (h *handler) get(w http.ResponseWriter, req *http.Request) {
	s, err := h.statuses.ByID(req.Context(), chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if s == nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("transfer not found"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(s))
}
