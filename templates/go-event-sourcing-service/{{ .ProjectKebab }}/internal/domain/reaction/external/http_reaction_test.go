//go:build unit

package external_test

import (
	"context"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/external"
	"github.com/stretchr/testify/require"
)

func TestHttpReaction(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("sends the event ID as the idempotency key", func(t *testing.T) {
			client := &recordingHttpClient{}
			reaction := external.NewHttpReaction("test-reaction", []string{"Event"}, "https://example.com/hook", "POST",
				func(event *domain.Event) ([]byte, error) {
					return []byte(`{}`), nil
				},
				client,
			)

			err := reaction.Handle(context.Background(), &domain.Event{ID: "3f2c9d1e-event-id"})

			require.NoError(t, err)
			require.Len(t, client.requests, 1)
			require.Equal(t, "3f2c9d1e-event-id", client.requests[0].Headers["Idempotency-Key"])
			require.Equal(t, "application/json", client.requests[0].Headers["Content-Type"])
		})
	})
}

type recordingHttpClient struct {
	requests []*external.HttpRequest
}

var _ external.HttpClient = (*recordingHttpClient)(nil)

func (c *recordingHttpClient) Do(req *external.HttpRequest) (*external.HttpResponse, error) {
	c.requests = append(c.requests, req)
	return &external.HttpResponse{StatusCode: 200}, nil
}
