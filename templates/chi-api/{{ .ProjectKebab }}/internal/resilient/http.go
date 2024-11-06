package resilient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/failsafehttp"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"
	"github.com/rs/zerolog"
)

func NewTimeoutPolicy(timeoutSeconds uint) timeout.Timeout[*http.Response] {
	return timeout.With[*http.Response](time.Duration(timeoutSeconds) * time.Second)
}

func NewRetryPolicy() retrypolicy.RetryPolicy[*http.Response] {
	return failsafehttp.RetryPolicyBuilder().
		WithBackoff(time.Second, 10*time.Second). // backoff with factor of 2, initial delay of 1s, stop if total delay is 10s
		WithMaxRetries(3).
		Build()
}

func NewCircuitBreakerPolicy(failureThreshold uint, successThreshold uint, logger zerolog.Logger) circuitbreaker.CircuitBreaker[*http.Response] {
	return circuitbreaker.Builder[*http.Response]().
		WithDelay(time.Minute).
		WithFailureThreshold(failureThreshold).
		WithSuccessThreshold(successThreshold).
		OnHalfOpen(func(_ circuitbreaker.StateChangedEvent) {
			logger.Info().Msg("circuit breaker half-opened")
		}).
		OnClose(func(_ circuitbreaker.StateChangedEvent) {
			logger.Info().Msg("circuit breaker closed")
		}).
		Build()
}

type apiResponse[T any] struct {
	Successful bool     `json:"Successful"`
	Messages   []string `json:"Messages"`
	Data       T        `json:"Data"`
}

// Get needs to be a normal function and not part of a struct in order to take advantage of generic
func Get[T any](httpClient *http.Client, token string, url string, failsafePolicies []failsafe.Policy[*http.Response]) (T, error) {
	var result T
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create get http request to %s: %v", url, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	failsafeRequest := failsafehttp.NewRequest(req, httpClient, failsafePolicies...)

	httpResponse, err := failsafeRequest.Do()
	if err != nil {
		if errors.Is(err, timeout.ErrExceeded) {
			httpClient.CloseIdleConnections()
			return result, fmt.Errorf("timeout exceeded when getting %s", url)
		}

		if errors.Is(err, circuitbreaker.ErrOpen) {
			return result, fmt.Errorf("circuit breaker open, url %s", url)
		}

		return result, fmt.Errorf("failed to get %s: %w", url, err)
	}

	apiResp, err := unmarshalResponseBody[T](httpResponse)
	if err != nil {
		return result, err
	}

	if !apiResp.Successful {
		return result, fmt.Errorf(strings.Join(apiResp.Messages, "\n"))
	}

	return apiResp.Data, nil
}

// Post needs to be a normal function and not part of a struct in order to take advantage of generic
func Post[T any](httpClient *http.Client, token string, url string, input interface{}, failsafePolicies []failsafe.Policy[*http.Response]) (T, error) {
	var result T
	marshalledInput, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal post input for %s: %v", url, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(marshalledInput))
	if err != nil {
		return result, fmt.Errorf("failed to create post http request to %s: %v", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	failsafeRequest := failsafehttp.NewRequest(req, httpClient, failsafePolicies...)

	httpResponse, err := failsafeRequest.Do()
	if err != nil {
		if errors.Is(err, timeout.ErrExceeded) {
			httpClient.CloseIdleConnections()
			return result, fmt.Errorf("timeout exceeded when posting %s", url)
		}

		if errors.Is(err, circuitbreaker.ErrOpen) {
			return result, fmt.Errorf("circuit breaker open, url %s", url)
		}

		return result, fmt.Errorf("failed to post %s: %v", url, err)
	}

	apiResp, err := unmarshalResponseBody[T](httpResponse)
	if err != nil {
		return result, err
	}

	if !apiResp.Successful {
		return result, fmt.Errorf(strings.Join(apiResp.Messages, "\n"))
	}

	return apiResp.Data, nil
}

func unmarshalResponseBody[T any](httpResponse *http.Response) (*apiResponse[T], error) {
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if !json.Valid(responseBody) {
		return nil, fmt.Errorf("response body is not valid json: %s", responseBody)
	}

	var apiResp apiResponse[T]
	if err := json.Unmarshal(responseBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v, status code %d, body: %s", err, httpResponse.StatusCode, string(responseBody))
	}

	return &apiResp, nil
}
