package test

import (
	"context"
	"log/slog"
)

func NewFakeLogHandler() *FakeLogHandler {
	return &FakeLogHandler{}
}

type FakeLogHandler struct {
	Record *slog.Record
	Attrs  []slog.Attr
	Group  string
}

func (l *FakeLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l *FakeLogHandler) Handle(ctx context.Context, record slog.Record) error {
	l.Record = &record
	return nil
}

func (l *FakeLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	l.Attrs = append(l.Attrs, attrs...)
	return l
}

func (l *FakeLogHandler) WithGroup(name string) slog.Handler {
	l.Group = name
	return l
}
