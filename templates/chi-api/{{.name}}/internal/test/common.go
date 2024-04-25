package test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/hpcsc/{{.name}}/internal/response"
	"github.com/stretchr/testify/require"
)

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
