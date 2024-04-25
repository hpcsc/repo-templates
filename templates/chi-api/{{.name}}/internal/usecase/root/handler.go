package root

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/unrolled/render"
)

var Version = "main"

func Register(router chi.Router) {
	h := &handler{
		renderer: render.New(),
	}
	router.Get("/", h.get)
}

type handler struct {
	renderer *render.Render
}

func (h *handler) get(w http.ResponseWriter, _ *http.Request) {
	_ = h.renderer.JSON(w, http.StatusOK, struct {
		Version string `json:"version"`
	}{
		Version: Version,
	})
}
