package open

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

var ErrNoOwner = errors.New("an account needs an owner")

type Report struct {
	AccountID string `json:"account_id"`
}

type Runner struct {
	store stream.Store
	now   func() time.Time
}

func New(store stream.Store, now func() time.Time) Runner {
	return Runner{store: store, now: now}
}

func (r Runner) Run(ctx context.Context, accountID, owner string) (Report, error) {
	err := stream.Update(ctx, r.store, account.StreamID(accountID), func(state *account.State) ([]es.Event, error) {
		return decide(state, accountID, owner, r.now())
	})
	if err != nil {
		return Report{}, fmt.Errorf("open account %q: %w", accountID, err)
	}
	return Report{AccountID: accountID}, nil
}

func decide(state *account.State, accountID, owner string, at time.Time) ([]es.Event, error) {
	if state.Open {
		return nil, nil
	}
	if owner == "" {
		return nil, ErrNoOwner
	}
	return []es.Event{event.AccountOpened{AccountID: accountID, Owner: owner, OpenedAt: at}}, nil
}
