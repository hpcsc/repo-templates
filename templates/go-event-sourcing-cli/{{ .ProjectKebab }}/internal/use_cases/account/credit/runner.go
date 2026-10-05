package credit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
)

var (
	ErrNotOpen       = errors.New("the account is not open")
	ErrInvalidAmount = errors.New("a credit must be more than 0")
)

type Report struct {
	AccountID string `json:"account_id"`
	Reference string `json:"reference"`
}

type Runner struct {
	store stream.Store
	now   func() time.Time
}

func New(store stream.Store, now func() time.Time) Runner {
	return Runner{store: store, now: now}
}

func (r Runner) Run(ctx context.Context, accountID string, amount int64, reference string) (Report, error) {
	err := stream.Update(ctx, r.store, account.StreamID(accountID), func(state *account.State) ([]es.Event, error) {
		return decide(state, accountID, amount, reference, r.now())
	})
	if err != nil {
		return Report{}, fmt.Errorf("credit account %q: %w", accountID, err)
	}
	return Report{AccountID: accountID, Reference: reference}, nil
}

func decide(state *account.State, accountID string, amount int64, reference string, at time.Time) ([]es.Event, error) {
	if !state.Open {
		return nil, ErrNotOpen
	}
	if state.Credited(reference) {
		return nil, nil
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	return []es.Event{event.AccountCredited{AccountID: accountID, Amount: amount, Reference: reference, CreditedAt: at}}, nil
}
