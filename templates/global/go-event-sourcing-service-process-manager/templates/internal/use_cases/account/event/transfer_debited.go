package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferDebited struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferDebited) EventType() string {
	return domainEvent.Namespace + ".transfer-debited"
}
