package outside

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultAttempts = 3
	DefaultWait     = 200 * time.Millisecond
)

type Option func(*Client)

func WithRetries(attempts int, wait time.Duration) Option {
	return func(c *Client) {
		c.attempts = attempts
		c.wait = wait
	}
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		http:     &http.Client{Timeout: 5 * time.Second},
		attempts: DefaultAttempts,
		wait:     DefaultWait,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Client struct {
	http     *http.Client
	attempts int
	wait     time.Duration
}

// KeyFor gives one key to all the requests of one call in one attempt of a payment, so an outside
// system answers a repeated request once.
func KeyFor(paymentID uuid.UUID, call string, attempt int) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s/%s/%d", paymentID, call, attempt))).String()
}

// Post sends the body as JSON and decodes the answer into answer, when answer is not nil. It tries
// again after a network error or a 5xx answer, and the outside system then gets the same key.
func (c *Client) Post(ctx context.Context, url string, key string, body any, answer any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= c.attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.wait * time.Duration(attempt-1)):
			}
		}

		retry, err := c.post(ctx, url, key, payload, answer)
		if err == nil || !retry {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("%w, after %d attempts", lastErr, c.attempts)
}

func (c *Client) post(ctx context.Context, url string, key string, payload []byte, answer any) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)

	resp, err := c.http.Do(req)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("failed to send POST %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return resp.StatusCode >= 500, fmt.Errorf("POST %s answered %d: %s", url, resp.StatusCode, bytes.TrimSpace(text))
	}
	if answer == nil {
		return false, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(answer); err != nil {
		return false, fmt.Errorf("failed to read the answer of POST %s: %w", url, err)
	}
	return false, nil
}
