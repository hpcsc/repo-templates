package aggregate

import (
	"errors"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
)

// Account is an example aggregate. Delete it along with the rest of the
// example slice once your own domain has one.
//
// The shape is the part worth keeping: private state, an Apply that folds
// history onto it, and one method per command that reads that state to decide
// and returns events without writing anything itself.
type Account struct {
	open     bool
	balance  int64
	credited map[string]bool
}

func NewAccount() *Account { return &Account{} }

func (a *Account) Apply(e es.Event) {
	switch applied := e.(type) {
	case event.AccountOpened:
		a.open = true

	case event.AccountCredited:
		a.balance += applied.Amount
		if a.credited == nil {
			a.credited = make(map[string]bool)
		}
		a.credited[applied.Reference] = true
	}
}

func (a *Account) Balance() int64 { return a.balance }

func (a *Account) OpenAccount(cmd command.OpenAccount) ([]es.Event, error) {
	// Deciding nothing is how an aggregate says "already done". The repository
	// writes no events, so opening the same account twice is not an error.
	if a.open {
		return nil, nil
	}

	if cmd.Owner == "" {
		return nil, errors.New("an account needs an owner")
	}

	return []es.Event{event.AccountOpened{
		AccountID: cmd.AccountID,
		Owner:     cmd.Owner,
		OpenedAt:  cmd.OpenedAt,
	}}, nil
}

func (a *Account) CreditAccount(cmd command.CreditAccount) ([]es.Event, error) {
	if !a.open {
		return nil, errors.New("account is not open")
	}

	// A retried credit must not add the amount twice, so the caller's own
	// reference is what makes the command safe to repeat.
	if a.credited[cmd.Reference] {
		return nil, nil
	}

	if cmd.Amount <= 0 {
		return nil, errors.New("a credit must be positive")
	}

	return []es.Event{event.AccountCredited{
		AccountID:  cmd.AccountID,
		Amount:     cmd.Amount,
		Reference:  cmd.Reference,
		CreditedAt: cmd.CreditedAt,
	}}, nil
}
