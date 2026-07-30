package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/aggregate"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type Report struct {
	AccountID string `json:"account_id"`
	Reference string `json:"reference"`
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

func (r *Runner) Run(ctx context.Context, accountID string, amount int64, reference string) (Report, error) {
	cmd := command.CreditAccount{
		AccountID:  accountID,
		Amount:     amount,
		Reference:  reference,
		CreditedAt: r.now(),
	}

	err := r.accounts.Update(ctx, accountID, func(a *aggregate.Account) ([]es.Event, error) {
		return a.CreditAccount(cmd)
	})
	if err != nil {
		return Report{}, fmt.Errorf("credit account %q: %w", accountID, err)
	}

	return Report{AccountID: accountID, Reference: reference}, nil
}
