package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferFailed struct {
	TransferID uuid.UUID `json:"transferId"`
	Reason     string    `json:"reason"`
}

func (TransferFailed) EventType() string {
	return domainEvent.Namespace + ".transfer-failed"
}
