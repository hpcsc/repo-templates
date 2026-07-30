// Package app is the composition root: the one place that knows how every
// other package is wired together. Tests build the same graph through New, so
// there is no second wiring that can drift from the real one.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/credit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/show"

	// Importing the event package is what registers its types with the codec.
	// Without it the store would fail to decode anything it reads back.
	_ "github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
)

type Config struct {
	DatabaseURL string
	Now         func() time.Time
}

type App struct {
	Open   *open.Runner
	Credit *credit.Runner
	Show   *show.Runner

	close func()
}

// New opens the store and builds every use case over it. Call Close when done.
func New(ctx context.Context, cfg Config) (*App, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("app: DatabaseURL is required")
	}

	postgres, err := store.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	return NewWithStore(postgres, cfg.Now, postgres.Close), nil
}

// NewWithStore builds the same graph over a store the caller already has,
// which is how unit tests run the real use cases against an in-memory store.
func NewWithStore(eventStore es.Store, now func() time.Time, close func()) *App {
	if now == nil {
		now = time.Now
	}
	if close == nil {
		close = func() {}
	}

	return &App{
		Open:   open.New(eventStore, now),
		Credit: credit.New(eventStore, now),
		Show:   show.New(eventStore),
		close:  close,
	}
}

func (a *App) Close() { a.close() }
