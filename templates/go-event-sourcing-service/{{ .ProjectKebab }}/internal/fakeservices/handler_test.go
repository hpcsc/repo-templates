//go:build unit

package fakeservices_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/fakeservices"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("fraud check", func(t *testing.T) {
		t.Run("passes an amount up to 10000", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			w := send(t, "/fraud/checks", "key-1", request(webhooks.URL, 10000, "DE89370400440532013000"))

			require.Equal(t, http.StatusAccepted, w.Code)
			require.Equal(t, `{"paymentId":"payment-1","passed":true}`, webhooks.waitForOne(t))
		})

		t.Run("fails an amount above 10000", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			send(t, "/fraud/checks", "key-1", request(webhooks.URL, 10001, "DE89370400440532013000"))

			require.Equal(t, `{"paymentId":"payment-1","passed":false,"reason":"amount above the limit of the fraud check"}`, webhooks.waitForOne(t))
		})
	})

	t.Run("sanctions screening", func(t *testing.T) {
		t.Run("fails a creditor IBAN that ends in 0000", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			send(t, "/sanctions/screenings", "key-1", request(webhooks.URL, 300, "DE89370400440532010000"))

			require.Equal(t, `{"paymentId":"payment-1","passed":false,"reason":"creditor on the sanctions list"}`, webhooks.waitForOne(t))
		})

		t.Run("never answers for a creditor IBAN that ends in 1111", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			w := send(t, "/sanctions/screenings", "key-1", request(webhooks.URL, 300, "DE89370400440532011111"))

			require.Equal(t, http.StatusAccepted, w.Code)
			require.Never(t, func() bool { return len(webhooks.bodies()) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
		})
	})

	t.Run("payment network", func(t *testing.T) {
		t.Run("gives a reference and settles the payment", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			w := send(t, "/network/payments", "key-1", request(webhooks.URL, 300, "DE89370400440532013000"))

			require.Equal(t, http.StatusAccepted, w.Code)
			require.Regexp(t, `^\{"reference":"N-[0-9A-F]{8}"\}\n$`, w.Body.String())
			require.Equal(t, `{"paymentId":"payment-1","status":"settled"}`, webhooks.waitForOne(t))
		})

		t.Run("returns the payment for a creditor IBAN that ends in 2222", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			send(t, "/network/payments", "key-1", request(webhooks.URL, 300, "DE89370400440532012222"))

			require.Equal(t, `{"paymentId":"payment-1","status":"returned","reason":"account closed at the other bank"}`, webhooks.waitForOne(t))
		})
	})

	t.Run("payment status", func(t *testing.T) {
		for iban, expected := range map[string]string{
			"DE89370400440532013000": `{"paymentId":"payment-1","status":"settled"}`,
			"DE89370400440532012222": `{"paymentId":"payment-1","status":"returned","reason":"account closed at the other bank"}`,
			"DE89370400440532013333": `{"paymentId":"payment-1","status":"settled"}`,
			"DE89370400440532014444": `{"paymentId":"payment-1","status":"pending"}`,
		} {
			t.Run("answers the status by the reference for a creditor IBAN that ends in "+iban[len(iban)-4:], func(t *testing.T) {
				handler := fakeservices.NewHandler("valid-token", time.Hour, test.DiscardLogger())
				submitted := sendTo(handler, "/network/payments", "key-1", request("http://unused", 300, iban))
				var answer fakeservices.SubmitAnswer
				require.NoError(t, json.Unmarshal(submitted.Body.Bytes(), &answer))

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest("GET", "/network/payments/"+answer.Reference, nil))

				require.Equal(t, http.StatusOK, w.Code)
				require.JSONEq(t, expected, w.Body.String())
			})
		}

		t.Run("sends no webhook for a creditor IBAN that ends in 3333", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			send(t, "/network/payments", "key-1", request(webhooks.URL, 300, "DE89370400440532013333"))

			require.Never(t, func() bool { return len(webhooks.bodies()) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
		})

		t.Run("answers not found for an unknown reference", func(t *testing.T) {
			w := httptest.NewRecorder()

			fakeservices.NewHandler("valid-token", 0, test.DiscardLogger()).ServeHTTP(w, httptest.NewRequest("GET", "/network/payments/N-UNKNOWN", nil))

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("repeated request", func(t *testing.T) {
		t.Run("gives the same answer and calls the webhook once", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()
			handler := fakeservices.NewHandler("valid-token", 0, test.DiscardLogger())

			first := sendTo(handler, "/network/payments", "key-1", request(webhooks.URL, 300, "DE89370400440532013000"))
			second := sendTo(handler, "/network/payments", "key-1", request(webhooks.URL, 300, "DE89370400440532013000"))

			require.Equal(t, first.Body.String(), second.Body.String())
			webhooks.waitForOne(t)
			require.Never(t, func() bool { return len(webhooks.bodies()) > 1 }, 200*time.Millisecond, 10*time.Millisecond)
		})

		t.Run("refuses a request with no Idempotency-Key", func(t *testing.T) {
			w := send(t, "/fraud/checks", "", request("http://unused", 300, "DE89370400440532013000"))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("webhook", func(t *testing.T) {
		t.Run("sends the token and the key of the request", func(t *testing.T) {
			webhooks := newWebhooks()
			defer webhooks.Close()

			send(t, "/fraud/checks", "key-1", request(webhooks.URL, 300, "DE89370400440532013000"))

			webhooks.waitForOne(t)
			require.Equal(t, []string{"Bearer valid-token key-1"}, webhooks.headers())
		})
	})
}

func request(webhookURL string, amount int64, creditorIBAN string) string {
	return fmt.Sprintf(`{"paymentId": "payment-1", "amount": %d, "creditorIban": %q, "webhookUrl": %q}`, amount, creditorIBAN, webhookURL)
}

func send(t *testing.T, path string, key string, body string) *httptest.ResponseRecorder {
	return sendTo(fakeservices.NewHandler("valid-token", 0, test.DiscardLogger()), path, key, body)
}

func sendTo(handler http.Handler, path string, key string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	handler.ServeHTTP(w, req)
	return w
}

type webhooks struct {
	*httptest.Server
	mutex    sync.Mutex
	received []string
	auth     []string
}

func newWebhooks() *webhooks {
	h := &webhooks{}
	h.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		h.mutex.Lock()
		h.received = append(h.received, string(body))
		h.auth = append(h.auth, r.Header.Get("Authorization")+" "+r.Header.Get("Idempotency-Key"))
		h.mutex.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	return h
}

func (h *webhooks) bodies() []string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return append([]string(nil), h.received...)
}

func (h *webhooks) headers() []string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return append([]string(nil), h.auth...)
}

func (h *webhooks) waitForOne(t *testing.T) string {
	require.Eventually(t, func() bool { return len(h.bodies()) > 0 }, 5*time.Second, 10*time.Millisecond, "no webhook arrived")
	return h.bodies()[0]
}
