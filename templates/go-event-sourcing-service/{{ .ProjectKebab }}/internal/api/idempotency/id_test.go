//go:build unit

package idempotency_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/idempotency"
	"github.com/stretchr/testify/require"
)

func TestIDFor(t *testing.T) {
	t.Run("id", func(t *testing.T) {
		t.Run("gives the same ID to two requests with the same key", func(t *testing.T) {
			first := withKey(httptest.NewRequest("POST", "/accounts", nil), "key-1")
			retry := withKey(httptest.NewRequest("POST", "/accounts", nil), "key-1")

			require.Equal(t, idempotency.IDFor(first, "command"), idempotency.IDFor(retry, "command"))
		})

		t.Run("gives a new ID to each request with no key", func(t *testing.T) {
			first := httptest.NewRequest("POST", "/accounts", nil)
			retry := httptest.NewRequest("POST", "/accounts", nil)

			require.NotEqual(t, idempotency.IDFor(first, "command"), idempotency.IDFor(retry, "command"))
		})

		t.Run("gives another ID to the same key on another path", func(t *testing.T) {
			deposit := withKey(httptest.NewRequest("POST", "/accounts/a/deposits", nil), "key-1")
			withdrawal := withKey(httptest.NewRequest("POST", "/accounts/a/withdrawals", nil), "key-1")

			require.NotEqual(t, idempotency.IDFor(deposit, "command"), idempotency.IDFor(withdrawal, "command"))
		})

		t.Run("gives another ID to the same request for another purpose", func(t *testing.T) {
			req := withKey(httptest.NewRequest("POST", "/accounts", nil), "key-1")

			require.NotEqual(t, idempotency.IDFor(req, "command"), idempotency.IDFor(req, "account"))
		})
	})
}

func withKey(req *http.Request, key string) *http.Request {
	req.Header.Set(idempotency.Header, key)
	return req
}
