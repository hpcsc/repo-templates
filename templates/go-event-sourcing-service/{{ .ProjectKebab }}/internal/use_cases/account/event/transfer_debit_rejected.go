package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferDebitRejected struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"`
}

func (TransferDebitRejected) EventType() string {
	return domainEvent.Namespace + ".transfer-debit-rejected"
}
