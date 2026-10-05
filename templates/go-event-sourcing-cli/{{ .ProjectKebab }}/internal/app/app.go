package app

import (
	"context"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/credit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/show"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/events"
)

type Config struct {
	Now func() time.Time
{{- if .Scaffold.SQLite }}

	StorePath string
{{- end }}
}

type App struct {
	Open   open.Runner
	Credit credit.Runner
	Show   show.Runner
	Events events.Runner

	close func() error
}

func New(ctx context.Context, cfg Config) (*App, error) {
	eventStore, closeStore, err := openStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &App{
		Open:   open.New(eventStore, now),
		Credit: credit.New(eventStore, now),
		Show:   show.New(eventStore),
		Events: events.New(eventStore),
		close:  closeStore,
	}, nil
}

func (a *App) Close() error {
	return a.close()
}

func openStore(ctx context.Context, cfg Config) (stream.Store, func() error, error) {
{{- if .Scaffold.SQLite }}
	if cfg.StorePath != "" {
		sqlite, err := store.OpenSQLite(ctx, cfg.StorePath)
		if err != nil {
			return nil, nil, err
		}
		return sqlite, sqlite.Close, nil
	}
{{- end }}
	return store.NewMemory(), func() error { return nil }, nil
}
