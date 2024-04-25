//go:build unit

package user

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{.name}}/internal/response"
	"github.com/hpcsc/{{.name}}/internal/test"
	"github.com/stretchr/testify/require"
)

func TestUserHandler(t *testing.T) {
	newRouter := func() *chi.Mux {
		router := chi.NewRouter()
		Register(router)
		return router
	}

	validPostRequest := func() *postRequest {
		return &postRequest{
			Name:  "test-user",
			Email: "test@example.com",
			Age:   20,
		}
	}

	t.Run("return Bad Request when request body is not valid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := newRouter()

		req, err := http.NewRequest("POST", "/users", strings.NewReader("invalid-json"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		resp := test.UnmarshalJson[response.Response](t, w.Body.Bytes())
		require.False(t, resp.Successful)

		test.RequireErrorResponse(t, w, "received invalid request body")
	})

	for _, tc := range []struct {
		scenario      string
		prepare       func(request *postRequest) *postRequest
		expectedError string
	}{
		{
			scenario: "name is missing",
			prepare: func(r *postRequest) *postRequest {
				r.Name = ""
				return r
			},
			expectedError: "Name is required",
		},
		{
			scenario: "name length is shorter than 7 characters",
			prepare: func(r *postRequest) *postRequest {
				r.Name = "test"
				return r
			},
			expectedError: "Name min length is 7",
		},
		{
			scenario: "invalid email",
			prepare: func(r *postRequest) *postRequest {
				r.Email = "test"
				return r
			},
			expectedError: "email is invalid",
		},
		{
			scenario: "age is missing",
			prepare: func(r *postRequest) *postRequest {
				r.Age = 0
				return r
			},
			expectedError: "Age is required",
		},
		{
			scenario: "age is less than 18",
			prepare: func(r *postRequest) *postRequest {
				r.Age = 17
				return r
			},
			expectedError: "age min value is 18",
		},
		{
			scenario: "age is more than 99",
			prepare: func(r *postRequest) *postRequest {
				r.Age = 100
				return r
			},
			expectedError: "age max value is 99",
		},
	} {
		t.Run(fmt.Sprintf("return Bad Request when %s", tc.scenario), func(t *testing.T) {
			w := httptest.NewRecorder()
			router := newRouter()

			req, err := http.NewRequest(
				"POST",
				"/users",
				bytes.NewReader(test.MarshalJson(t, *tc.prepare(validPostRequest()))),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			resp := test.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.False(t, resp.Successful)

			test.RequireErrorResponse(t, w, tc.expectedError)
		})
	}

	t.Run("return Ok when successful", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := newRouter()

		req, err := http.NewRequest("POST", "/users", bytes.NewReader(test.MarshalJson(t, validPostRequest())))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		resp := test.UnmarshalJson[response.Response](t, w.Body.Bytes())
		require.True(t, resp.Successful)
	})
}
