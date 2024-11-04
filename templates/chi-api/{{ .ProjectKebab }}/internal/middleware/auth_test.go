package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	newRouter := func(t *testing.T) chi.Router {
		router := chi.NewRouter()

		authMiddleware, err := NewAuthMiddleware("testdata/valid-token")
		require.NoError(t, err)

		router.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("success"))
			})
		})

		return router
	}

	t.Run("return error when unable to read token file", func(t *testing.T) {
		_, err := NewAuthMiddleware("invalid-path")
		require.ErrorContains(t, err, "failed to read token")
	})

	t.Run("return Forbidden when token from Authorization header not matches", func(t *testing.T) {
		router := newRouter(t)
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/protected", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("return Ok when token from Authorization header matches", func(t *testing.T) {
		router := newRouter(t)
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/protected", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer valid-token")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "success", w.Body.String())
	})
}
