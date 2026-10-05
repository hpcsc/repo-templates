package account

import (
	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/account/event"
)

type transferLedger struct {
	debited  map[uuid.UUID]bool
	credited map[uuid.UUID]bool
	refunded map[uuid.UUID]bool
}

func (a *Account) DebitForTransfer(transferID uuid.UUID, amount int64) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if a.transfers.debited[transferID] {
		return nil
	}

	if err := a.checkDebit(amount); err != nil {
		a.record(domain.NewEvent(&event.TransferDebitRejected{
			AccountID:  a.id,
			TransferID: transferID,
			Amount:     amount,
			Reason:     err.Error(),
		}))
		return nil
	}

	a.record(domain.NewEvent(&event.TransferDebited{
		AccountID:  a.id,
		TransferID: transferID,
		Amount:     amount,
	}))
	return nil
}

func (a *Account) CreditForTransfer(transferID uuid.UUID, amount int64) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if a.transfers.credited[transferID] {
		return nil
	}

	if a.closed {
		a.record(domain.NewEvent(&event.TransferCreditRejected{
			AccountID:  a.id,
			TransferID: transferID,
			Amount:     amount,
			Reason:     ErrClosed.Error(),
		}))
		return nil
	}

	a.record(domain.NewEvent(&event.TransferCredited{
		AccountID:  a.id,
		TransferID: transferID,
		Amount:     amount,
	}))
	return nil
}

func (a *Account) RefundTransfer(transferID uuid.UUID, amount int64) error {
	if a.id == uuid.Nil {
		return ErrNotOpened
	}
	if a.transfers.refunded[transferID] {
		return nil
	}

	a.record(domain.NewEvent(&event.TransferRefunded{
		AccountID:  a.id,
		TransferID: transferID,
		Amount:     amount,
	}))
	return nil
}

func (a *Account) applyTransfer(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.TransferDebited:
		a.balance -= p.Amount
		a.transfers.debited = marked(a.transfers.debited, p.TransferID)
	case *event.TransferDebitRejected:
		a.transfers.debited = marked(a.transfers.debited, p.TransferID)
	case *event.TransferCredited:
		a.balance += p.Amount
		a.transfers.credited = marked(a.transfers.credited, p.TransferID)
	case *event.TransferCreditRejected:
		a.transfers.credited = marked(a.transfers.credited, p.TransferID)
	case *event.TransferRefunded:
		a.balance += p.Amount
		a.transfers.refunded = marked(a.transfers.refunded, p.TransferID)
	}
}

func (a *Account) checkDebit(amount int64) error {
	if err := a.checkChange(amount); err != nil {
		return err
	}
	if amount > a.balance {
		return ErrInsufficientFunds
	}
	return nil
}

func marked(transfers map[uuid.UUID]bool, transferID uuid.UUID) map[uuid.UUID]bool {
	if transfers == nil {
		transfers = make(map[uuid.UUID]bool)
	}
	transfers[transferID] = true
	return transfers
}
