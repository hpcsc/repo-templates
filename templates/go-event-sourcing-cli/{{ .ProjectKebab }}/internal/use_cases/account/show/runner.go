package show

import (
	"context"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
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

func (r *Report) Apply(record es.Record) {
	switch e := record.Event.(type) {
	case event.AccountOpened:
		r.AccountID = e.AccountID
		r.Owner = e.Owner
		r.OpenedAt = e.OpenedAt
	case event.AccountCredited:
		r.Balance += e.Amount
		r.Credits = append(r.Credits, Credit{Amount: e.Amount, Reference: e.Reference, CreditedAt: e.CreditedAt})
	}
}

type Runner struct {
	store stream.Store
}

func New(store stream.Store) Runner {
	return Runner{store: store}
}

func (r Runner) Run(ctx context.Context, accountID string) (Report, bool, error) {
	history, _, err := r.store.Load(ctx, account.StreamID(accountID))
	if err != nil {
		return Report{}, false, fmt.Errorf("show account %q: %w", accountID, err)
	}
	if len(history) == 0 {
		return Report{}, false, nil
	}

	report := Report{Credits: []Credit{}}
	stream.Fold(history, &report)
	return report, true, nil
}
