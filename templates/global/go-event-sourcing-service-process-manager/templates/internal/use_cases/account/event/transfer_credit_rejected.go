package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferCreditRejected struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"`
}

func (TransferCreditRejected) EventType() string {
	return domainEvent.Namespace + ".transfer-credit-rejected"
}
