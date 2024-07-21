package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hpcsc/{{.name}}/internal/middleware/request"
	"github.com/hpcsc/{{.name}}/internal/test"
	"github.com/stretchr/testify/require"
)

const (
	requestID = "test-request-id"
)

func TestRequestLogger(t *testing.T) {
	type TestCase struct {
		level slog.Level
		code  int
	}

	for _, c := range []TestCase{
		{slog.LevelInfo, http.StatusOK},
		{slog.LevelInfo, http.StatusTemporaryRedirect},
		{slog.LevelWarn, http.StatusBadRequest},
		{slog.LevelError, http.StatusInternalServerError},
	} {
		t.Run(fmt.Sprintf("logs request and response fields for routes with a %d response", c.code), func(t *testing.T) {
			logHandler := test.NewFakeLogHandler()
			logger := slog.New(logHandler)
			now := time.Now()
			logMiddleware := RequestLogger(logger, request.NewFakeClock(t, now, now), nil)

			w := httptest.NewRecorder()
			router := testRouterWithRequestLogger("/test/route", c.code, logMiddleware)

			httpRequest, err := http.NewRequest("GET", "/test/route", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, httpRequest)

			requestFields := requestFieldsFrom(httpRequest)
			responseFields := responseFieldsWithStatus(c.code, 0)

			message := fmt.Sprintf("[%d] GET /test/route", c.code)

			require.Equal(t, c.code, w.Code)
			require.NotNil(t, logHandler.Record)
			require.Equal(t, message, logHandler.Record.Message)
			requireRequestAndResponseFields(t, logHandler, requestFields, responseFields)
		})
	}

	t.Run("does not log for ignored routes", func(t *testing.T) {
		logHandler := test.NewFakeLogHandler()
		logger := slog.New(logHandler)
		now := time.Now()
		logMiddleware := RequestLogger(logger, request.NewFakeClock(t, now, now), []string{
			"/ignored/route",
		})

		w := httptest.NewRecorder()
		router := testRouterWithRequestLogger("/ignored/route", http.StatusOK, logMiddleware)

		httpRequest, err := http.NewRequest("GET", "/ignored/route", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, httpRequest)

		require.Equal(t, http.StatusOK, w.Code)
		require.Nil(t, logHandler.Record)
	})

	t.Run("still logs for ignored routes with an error response", func(t *testing.T) {
		logHandler := test.NewFakeLogHandler()
		logger := slog.New(logHandler)
		now := time.Now()
		logMiddleware := RequestLogger(logger, request.NewFakeClock(t, now, now), []string{
			"/ignored/route",
		})

		w := httptest.NewRecorder()
		router := testRouterWithRequestLogger("/ignored/route", http.StatusInternalServerError, logMiddleware)

		httpRequest, err := http.NewRequest("GET", "/ignored/route", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, httpRequest)

		requestFields := requestFieldsFrom(httpRequest)
		responseFields := responseFieldsWithStatus(500, 0)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.NotNil(t, logHandler.Record)
		require.Equal(t, "[500] GET /ignored/route", logHandler.Record.Message)
		requireRequestAndResponseFields(t, logHandler, requestFields, responseFields)
	})

	t.Run("still logs for ignored routes with duration longer than threshold", func(t *testing.T) {
		logHandler := test.NewFakeLogHandler()
		logger := slog.New(logHandler)
		now := time.Now()
		logMiddleware := RequestLogger(logger, request.NewFakeClock(t, now, now.Add(longRequestThreshold+1*time.Millisecond)), []string{
			"/ignored/route",
		})

		w := httptest.NewRecorder()
		router := testRouterWithRequestLogger("/ignored/route", http.StatusOK, logMiddleware)

		httpRequest, err := http.NewRequest("GET", "/ignored/route", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, httpRequest)

		requestFields := requestFieldsFrom(httpRequest)
		responseFields := responseFieldsWithStatus(200, 501)

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, logHandler.Record)
		require.Equal(t, "[200] GET /ignored/route", logHandler.Record.Message)
		requireRequestAndResponseFields(t, logHandler, requestFields, responseFields)
	})
}

func responseFieldsWithStatus(status int, elapsed float64) slog.Attr {
	return slog.Attr{
		Key: "response", Value: slog.GroupValue(
			slog.Attr{Key: "status", Value: slog.IntValue(status)},
			slog.Attr{Key: "elapsed", Value: slog.Float64Value(elapsed)},
		),
	}
}

func requestFieldsFrom(httpRequest *http.Request) slog.Attr {
	return slog.Attr{
		Key: "request", Value: slog.GroupValue(
			slog.Attr{Key: "url", Value: slog.StringValue("http://")},
			slog.Attr{Key: "method", Value: slog.StringValue(httpRequest.Method)},
			slog.Attr{Key: "path", Value: slog.StringValue(httpRequest.URL.Path)},
			slog.Attr{Key: "ip", Value: slog.StringValue(httpRequest.RemoteAddr)},
			slog.Attr{Key: "proto", Value: slog.StringValue(httpRequest.Proto)},
			slog.Attr{Key: "id", Value: slog.StringValue(requestID)},
		),
	}
}

func requireRequestAndResponseFields(t *testing.T, logHandler *test.FakeLogHandler, requestFields slog.Attr, responseFields slog.Attr) {
	require.Len(t, logHandler.Attrs, 1)
	httpFields := logHandler.Attrs[0].Value.Group()
	if httpFields[0].Key == "request" {
		require.Truef(t, requestFields.Equal(httpFields[0]), fmt.Sprintf("expected %v, but got %v", requestFields, httpFields[0]))
		require.True(t, responseFields.Equal(httpFields[1]), fmt.Sprintf("expected %v, but got %v", responseFields, httpFields[1]))
	} else {
		require.Truef(t, requestFields.Equal(httpFields[1]), fmt.Sprintf("expected %v, but got %v", requestFields, httpFields[1]))
		require.True(t, responseFields.Equal(httpFields[0]), fmt.Sprintf("expected %v, but got %v", responseFields, httpFields[0]))
	}
}

func testRouterWithRequestLogger(routePattern string, statusCode int, logMiddleware func(next http.Handler) http.Handler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(staticRequestID)
	router.Use(logMiddleware)
	router.Get(routePattern, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	})

	return router
}

func staticRequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, middleware.RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
