package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferCreditRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferCreditRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-credit-requested"
}
