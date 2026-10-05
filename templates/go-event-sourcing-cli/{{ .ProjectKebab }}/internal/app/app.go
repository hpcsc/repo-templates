package app

import (
	"context"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/credit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/show"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/events"
)

type Config struct {
	Now       func() time.Time
	StorePath string
}

type App struct {
	Open   open.Runner
	Credit credit.Runner
	Show   show.Runner
	Events events.Runner

	close func() error
}

func New(ctx context.Context, cfg Config) (*App, error) {
	eventStore, err := store.OpenSQLite(ctx, cfg.StorePath)
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
		close:  eventStore.Close,
	}, nil
}

func (a *App) Close() error {
	return a.close()
}
