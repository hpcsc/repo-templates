package transfer

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
)

var (
	ErrAlreadyStarted = errors.New("transfer is already started")
	ErrSameAccount    = errors.New("transfer must be between two different accounts")
	ErrInvalidAmount  = errors.New("amount must be more than 0")
	ErrUnknownAccount = errors.New("account does not exist")
)

var _ domain.Aggregate = (*Transfer)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("transfer-%s", id)
}

type Transfer struct {
	domain.AggregateBase

	id uuid.UUID
}

func (t *Transfer) ID() uuid.UUID {
	return t.id
}

func (t *Transfer) Start(id uuid.UUID, from uuid.UUID, to uuid.UUID, amount int64) error {
	if t.id != uuid.Nil {
		return ErrAlreadyStarted
	}
	if from == to {
		return ErrSameAccount
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}

	t.record(domain.NewEvent(&event.TransferStarted{
		TransferID: id,
		From:       from,
		To:         to,
		Amount:     amount,
	}))
	return nil
}

func (t *Transfer) Apply(e domain.Event) {
	if started, ok := e.Payload.(*event.TransferStarted); ok {
		t.id = started.TransferID
	}
}

func (t *Transfer) record(e domain.Event) {
	t.Apply(e)
	t.Record(e)
}
