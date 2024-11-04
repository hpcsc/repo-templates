package request

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func NewLogEntry(r *http.Request, l *slog.Logger) middleware.LogEntry {
	return &LogEntry{
		request: r,
		logger:  l,
	}
}

type LogEntry struct {
	logger       *slog.Logger
	panicMessage string

	request *http.Request
}

var _ middleware.LogEntry = (*LogEntry)(nil)

func (l *LogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	logLevel := statusLevel(status)

	msg := l.message(status)
	fields := slog.Attr{
		Key: "http",
		Value: slog.GroupValue(
			requestLogFields(l.request),
			responseLogFields(status, elapsed),
		),
	}

	l.logger.With(fields).Log(context.Background(), logLevel, msg)
}

func (l *LogEntry) Panic(v interface{}, stack []byte) {
	l.logger = l.logger.With(
		slog.Attr{Key: "stacktrace", Value: slog.StringValue(string(stack))},
		slog.Attr{Key: "panic", Value: slog.StringValue(fmt.Sprintf("%+v", v))},
	)

	l.panicMessage = fmt.Sprintf("%+v", v)
}

func (l *LogEntry) message(status int) string {
	msg := fmt.Sprintf("[%d] %s %s", status, l.request.Method, l.request.URL.Path)

	if l.panicMessage != "" {
		msg = fmt.Sprintf("%s - %s", msg, l.panicMessage)
	}

	return msg
}

func requestLogFields(r *http.Request) slog.Attr {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	requestURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	requestFields := []slog.Attr{
		{Key: "url", Value: slog.StringValue(requestURL)},
		{Key: "method", Value: slog.StringValue(r.Method)},
		{Key: "path", Value: slog.StringValue(r.URL.Path)},
		{Key: "ip", Value: slog.StringValue(r.RemoteAddr)},
		{Key: "proto", Value: slog.StringValue(r.Proto)},
	}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		requestFields = append(requestFields, slog.Attr{Key: "id", Value: slog.StringValue(reqID)})
	}

	return slog.Attr{Key: "request", Value: slog.GroupValue(requestFields...)}
}

func responseLogFields(status int, elapsed time.Duration) slog.Attr {
	fields := []slog.Attr{
		{Key: "status", Value: slog.IntValue(status)},
		{Key: "elapsed", Value: slog.Float64Value(float64(elapsed.Nanoseconds()) / 1000000.0)}, // in milliseconds
	}

	return slog.Attr{Key: "response", Value: slog.GroupValue(fields...)}
}

func statusLevel(status int) slog.Level {
	switch {
	case status < 400: // for codes in 100s, 200s, 300s
		return slog.LevelInfo
	case status < 500:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
