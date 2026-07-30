// Package show is the read side. It never loads an aggregate: aggregates
// exist to decide, and shaping a view is not a decision.
package show

import (
	"context"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type Credit struct {
	Amount     int64     `json:"amount"`
	Reference  string    `json:"reference"`
	CreditedAt time.Time `json:"credited_at"`
}

type Report struct {
	AccountID string    `json:"account_id"`
	Owner     string    `json:"owner"`
	Balance   int64     `json:"balance"`
	OpenedAt  time.Time `json:"opened_at"`
	Credits   []Credit  `json:"credits"`
}

type Runner struct {
	store es.Store
}

func New(store es.Store) *Runner {
	return &Runner{store: store}
}

// Run folds the stream on read, which is the simplest thing that works and is
// fine while a stream is short. Once it is not, keep this signature and feed
// it from a projection table instead — callers do not need to know.
func (r *Runner) Run(ctx context.Context, accountID string) (Report, bool, error) {
	history, _, err := r.store.Load(ctx, accountID)
	if err != nil {
		return Report{}, false, fmt.Errorf("show account %q: %w", accountID, err)
	}
	if len(history) == 0 {
		return Report{}, false, nil
	}

	var projection accountProjection
	projection.apply(history)

	return projection.report, true, nil
}
