package root

import (
	"net/http"

	"github.com/hpcsc/{{.ProjectKebab}}/internal/route"
	"github.com/unrolled/render"
)

var Version = "main"

var _ route.Routable = (*handler)(nil)

func NewHandler() route.Routable {
	return &handler{
		renderer: render.New(),
	}
}

type handler struct {
	renderer *render.Render
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Public("GET", "/", h.get),
	}
}

func (h *handler) get(w http.ResponseWriter, _ *http.Request) {
	_ = h.renderer.JSON(w, http.StatusOK, struct {
		Version string `json:"version"`
	}{
		Version: Version,
	})
}
