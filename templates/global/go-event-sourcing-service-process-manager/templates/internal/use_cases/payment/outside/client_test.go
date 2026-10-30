//go:build unit

package outside_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/outside"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		t.Run("reads the answer after a 5xx answer", func(t *testing.T) {
			server := newServer(http.StatusServiceUnavailable, http.StatusOK)
			defer server.Close()
			var answer struct {
				Reference string `json:"reference"`
			}

			err := newClient().Get(context.Background(), server.URL, &answer)

			require.NoError(t, err)
			require.Equal(t, "N-1", answer.Reference)
		})
	})

	t.Run("post", func(t *testing.T) {
		t.Run("sends the body as JSON with the key, and reads the answer", func(t *testing.T) {
			server := newServer(http.StatusAccepted)
			defer server.Close()
			var answer struct {
				Reference string `json:"reference"`
			}

			err := newClient().Post(context.Background(), server.URL, "key-1", map[string]int{"amount": 300}, &answer)

			require.NoError(t, err)
			require.Equal(t, []received{
				{key: "key-1", body: map[string]any{"amount": float64(300)}},
			}, server.requests())
			require.Equal(t, "N-1", answer.Reference)
		})

		t.Run("sends the same key again after a 5xx answer", func(t *testing.T) {
			server := newServer(http.StatusServiceUnavailable, http.StatusAccepted)
			defer server.Close()

			err := newClient().Post(context.Background(), server.URL, "key-1", map[string]int{"amount": 300}, nil)

			require.NoError(t, err)
			require.Equal(t, []string{"key-1", "key-1"}, server.keys())
		})

		t.Run("returns the error of a 4xx answer at once", func(t *testing.T) {
			server := newServer(http.StatusBadRequest, http.StatusAccepted)
			defer server.Close()

			err := newClient().Post(context.Background(), server.URL, "key-1", map[string]int{"amount": 300}, nil)

			require.ErrorContains(t, err, "answered 400")
			require.Equal(t, []string{"key-1"}, server.keys())
		})

		t.Run("returns the error after the last attempt", func(t *testing.T) {
			server := newServer(http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusServiceUnavailable)
			defer server.Close()

			err := newClient().Post(context.Background(), server.URL, "key-1", map[string]int{"amount": 300}, nil)

			require.ErrorContains(t, err, "after 3 attempts")
			require.Equal(t, []string{"key-1", "key-1", "key-1"}, server.keys())
		})
	})
}

func newClient() *outside.Client {
	return outside.NewClient(outside.WithRetries(3, time.Millisecond))
}

type received struct {
	key  string
	body map[string]any
}

type server struct {
	*httptest.Server
	mutex    sync.Mutex
	statuses []int
	received []received
}

// newServer answers each request with the next status, and with the last status after that.
func newServer(statuses ...int) *server {
	s := &server{statuses: statuses}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.received = append(s.received, received{key: r.Header.Get("Idempotency-Key"), body: body})
		status := s.statuses[min(len(s.received), len(s.statuses))-1]
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"reference": "N-1"}`))
	}))
	return s
}

func (s *server) requests() []received {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]received(nil), s.received...)
}

func (s *server) keys() []string {
	var keys []string
	for _, r := range s.requests() {
		keys = append(keys, r.key)
	}
	return keys
}
