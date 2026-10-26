package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferStarted struct {
	TransferID uuid.UUID `json:"transferId"`
	From       uuid.UUID `json:"from"`
	To         uuid.UUID `json:"to"`
	Amount     int64     `json:"amount"`
}

func (TransferStarted) EventType() string {
	return domainEvent.Namespace + ".transfer-started"
}
