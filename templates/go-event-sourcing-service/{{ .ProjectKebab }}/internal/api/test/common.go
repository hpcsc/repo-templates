package test

import (
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/response"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
	commonTest "github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/stretchr/testify/require"
)

func NewRouterWithRoutable(routable route.Routable) *chi.Mux {
	router := chi.NewRouter()
	for _, r := range routable.Routes() {
		// register all as public
		// auth middleware should be tested separately
		router.MethodFunc(r.Method, r.Pattern, r.Handler)
	}
	return router
}

func RequireErrorResponse(t *testing.T, w *httptest.ResponseRecorder, errorPattern string) {
	errs := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())

	require.False(t, errs.Successful)
	require.Len(t, errs.Messages, 1)
	require.Contains(t, errs.Messages[0], errorPattern)
}
