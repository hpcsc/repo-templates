package external

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
)

type HttpReaction struct {
	name           string
	eventTypes     []string
	url            string
	method         string
	payloadFactory func(event *domain.Event) ([]byte, error)
	httpClient     HttpClient
}

type HttpClient interface {
	Do(req *HttpRequest) (*HttpResponse, error)
}

type HttpRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

type HttpResponse struct {
	StatusCode int
	Body       []byte
}

func NewHttpReaction(
	name string,
	eventTypes []string,
	url string,
	method string,
	payloadFactory func(event *domain.Event) ([]byte, error),
	httpClient HttpClient,
) reaction.Interface {
	return &HttpReaction{
		name:           name,
		eventTypes:     eventTypes,
		url:            url,
		method:         method,
		payloadFactory: payloadFactory,
		httpClient:     httpClient,
	}
}

func (h *HttpReaction) Handle(ctx context.Context, event *domain.Event) error {
	payload, err := h.payloadFactory(event)
	if err != nil {
		return err
	}

	req := &HttpRequest{
		Method: h.method,
		URL:    h.url,
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Idempotency-Key": event.ID,
		},
		Body: payload,
	}

	_, err = h.httpClient.Do(req)
	return err
}

func (h *HttpReaction) EventTypes() []string {
	return h.eventTypes
}

func (h *HttpReaction) Name() string {
	return h.name
}
