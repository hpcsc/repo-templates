// Package open is one use case: everything needed to serve a single request,
// in one package. A new use case is a new folder like this one, not a new
// method on a growing service.
package open

import (
	"context"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/aggregate"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

// Report is what the caller gets back. Keeping it a type rather than a bare
// value leaves room to add fields without changing every call site.
type Report struct {
	AccountID string `json:"account_id"`
}

type Runner struct {
	accounts es.Repository[aggregate.Account, *aggregate.Account]
	now      func() time.Time
}

func New(store es.Store, now func() time.Time) *Runner {
	return &Runner{
		accounts: es.NewRepository[aggregate.Account](store, es.DefaultRetryAttempts),
		now:      now,
	}
}

func (r *Runner) Run(ctx context.Context, accountID, owner string) (Report, error) {
	cmd := command.OpenAccount{
		AccountID: accountID,
		Owner:     owner,
		OpenedAt:  r.now(),
	}

	err := r.accounts.Update(ctx, accountID, func(a *aggregate.Account) ([]es.Event, error) {
		return a.OpenAccount(cmd)
	})
	if err != nil {
		return Report{}, fmt.Errorf("open account %q: %w", accountID, err)
	}

	return Report{AccountID: accountID}, nil
}
