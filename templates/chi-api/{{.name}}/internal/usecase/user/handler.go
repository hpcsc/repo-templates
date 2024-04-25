package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gookit/validate"
	"github.com/hpcsc/{{.name}}/internal/response"
	"github.com/unrolled/render"
)

func Register(router chi.Router) {
	h := &handler{
		renderer: render.New(),
	}
	router.Post("/users", h.post)
}

type handler struct {
	renderer *render.Render
}

func (h *handler) post(w http.ResponseWriter, req *http.Request) {
	var postData postRequest
	if err := json.NewDecoder(req.Body).Decode(&postData); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return
	}

	v := validate.Struct(postData)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return
	}

	// do something with postData

	_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
}
