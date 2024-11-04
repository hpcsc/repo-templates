package user

import (
	"encoding/json"
	"net/http"

	"github.com/gookit/validate"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/response"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/route"
	"github.com/unrolled/render"
)

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
		route.Protected("POST", "/users", h.post),
	}
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
