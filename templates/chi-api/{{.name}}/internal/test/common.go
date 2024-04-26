package test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{.name}}/internal/response"
	"github.com/hpcsc/{{.name}}/internal/route"
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
	errs := UnmarshalJson[response.Response](t, w.Body.Bytes())

	require.False(t, errs.Successful)
	require.Len(t, errs.Messages, 1)
	require.Contains(t, errs.Messages[0], errorPattern)
}

func MarshalJson[T any](t *testing.T, input T) []byte {
	marshalled, err := json.Marshal(input)
	require.NoError(t, err)
	return marshalled
}

func UnmarshalJson[T any](t *testing.T, j []byte) *T {
	var unmarshalled T
	require.NoError(t, json.Unmarshal(j, &unmarshalled))
	return &unmarshalled
}
