package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferDebitRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferDebitRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-debit-requested"
}
