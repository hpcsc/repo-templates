package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferCompleted struct {
	TransferID uuid.UUID `json:"transferId"`
}

func (TransferCompleted) EventType() string {
	return domainEvent.Namespace + ".transfer-completed"
}
