package account

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
)

var ErrNoHold = errors.New("account has no hold for the payment")

type holdState int

const (
	holdPlaced holdState = iota + 1
	holdRejected
	holdReleased
	holdCaptured
)

type hold struct {
	amount int64
	state  holdState
}

func (a *Account) PlaceHold(paymentID uuid.UUID, amount int64) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if _, decided := a.holds[paymentID]; decided {
		return nil
	}

	if err := a.checkHold(amount); err != nil {
		a.record(domain.NewEvent(&event.HoldRejected{
			AccountID: a.id,
			PaymentID: paymentID,
			Amount:    amount,
			Reason:    err.Error(),
		}))
		return nil
	}

	a.record(domain.NewEvent(&event.HoldPlaced{
		AccountID: a.id,
		PaymentID: paymentID,
		Amount:    amount,
	}))
	return nil
}

func (a *Account) ReleaseHold(paymentID uuid.UUID) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	h, ok := a.holds[paymentID]
	if !ok || h.state != holdPlaced {
		return nil
	}

	a.record(domain.NewEvent(&event.HoldReleased{
		AccountID: a.id,
		PaymentID: paymentID,
		Amount:    h.amount,
	}))
	return nil
}

func (a *Account) CaptureHold(paymentID uuid.UUID) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	h, ok := a.holds[paymentID]
	if ok && h.state == holdCaptured {
		return nil
	}
	if !ok || h.state != holdPlaced {
		return fmt.Errorf("%w: %s", ErrNoHold, paymentID)
	}

	a.record(domain.NewEvent(&event.HoldCaptured{
		AccountID: a.id,
		PaymentID: paymentID,
		Amount:    h.amount,
	}))
	return nil
}

func (a *Account) applyHold(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.HoldPlaced:
		a.available -= p.Amount
		a.holds = withHold(a.holds, p.PaymentID, hold{amount: p.Amount, state: holdPlaced})
	case *event.HoldRejected:
		a.holds = withHold(a.holds, p.PaymentID, hold{amount: p.Amount, state: holdRejected})
	case *event.HoldReleased:
		a.available += p.Amount
		a.holds = withHold(a.holds, p.PaymentID, hold{amount: p.Amount, state: holdReleased})
	case *event.HoldCaptured:
		a.holds = withHold(a.holds, p.PaymentID, hold{amount: p.Amount, state: holdCaptured})
	}
}

func (a *Account) checkHold(amount int64) error {
	if err := a.checkChange(amount); err != nil {
		return err
	}
	if amount > a.available {
		return ErrInsufficientFunds
	}
	return nil
}

func withHold(holds map[uuid.UUID]hold, paymentID uuid.UUID, h hold) map[uuid.UUID]hold {
	if holds == nil {
		holds = make(map[uuid.UUID]hold)
	}
	holds[paymentID] = h
	return holds
}
