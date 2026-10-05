package account

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
)

var (
	ErrAlreadyOpened     = errors.New("account is already open")
	ErrNotOpened         = errors.New("account is not open")
	ErrInvalidAmount     = errors.New("amount must be more than 0")
	ErrInsufficientFunds = errors.New("balance is less than the amount")
	ErrClosed            = errors.New("account is closed")
)

var _ domain.Aggregate = (*Account)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("account-%s", id)
}

type Account struct {
	domain.AggregateBase

	id                  uuid.UUID
	balance             int64
	welcomeBonusGranted bool
	closed              bool
{{- if .Scaffold.ProcessManager }}
	transfers           transferLedger
{{- end }}
}

func (a *Account) ID() uuid.UUID {
	return a.id
}

func (a *Account) Open(id uuid.UUID, owner string) error {
	if a.id != uuid.Nil {
		return ErrAlreadyOpened
	}
	if owner == "" {
		return errors.New("owner must not be empty")
	}

	a.record(domain.NewEvent(&event.AccountOpened{
		AccountID: id,
		Owner:     owner,
	}))
	return nil
}

func (a *Account) Deposit(amount int64) error {
	if err := a.checkChange(amount); err != nil {
		return err
	}

	a.record(domain.NewEvent(&event.MoneyDeposited{
		AccountID: a.id,
		Amount:    amount,
	}))
	return nil
}

func (a *Account) Withdraw(amount int64) error {
	if err := a.checkChange(amount); err != nil {
		return err
	}
	if amount > a.balance {
		return fmt.Errorf("%w: balance %d, amount %d", ErrInsufficientFunds, a.balance, amount)
	}

	a.record(domain.NewEvent(&event.MoneyWithdrawn{
		AccountID: a.id,
		Amount:    amount,
	}))
	return nil
}

func (a *Account) GrantWelcomeBonus(amount int64) error {
	if a.closed {
		return nil
	}
	if err := a.checkChange(amount); err != nil {
		return err
	}
	if a.welcomeBonusGranted {
		return nil
	}

	a.record(domain.NewEvent(&event.WelcomeBonusGranted{
		AccountID: a.id,
		Amount:    amount,
	}))
	return nil
}

func (a *Account) Close() error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if a.closed {
		return ErrClosed
	}

	a.record(domain.NewEvent(&event.AccountClosed{AccountID: a.id}))
	return nil
}

func (a *Account) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.AccountOpened:
		a.id = p.AccountID
	case *event.MoneyDeposited:
		a.balance += p.Amount
	case *event.MoneyWithdrawn:
		a.balance -= p.Amount
	case *event.WelcomeBonusGranted:
		a.balance += p.Amount
		a.welcomeBonusGranted = true
	case *event.AccountClosed:
		a.closed = true
{{- if .Scaffold.ProcessManager }}
	default:
		a.applyTransfer(e)
{{- end }}
	}
}

func (a *Account) checkChange(amount int64) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if a.closed {
		return ErrClosed
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

func (a *Account) record(e domain.Event) {
	a.Apply(e)
	a.Record(e)
}
