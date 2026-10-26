package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferRefundRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferRefundRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-refund-requested"
}
