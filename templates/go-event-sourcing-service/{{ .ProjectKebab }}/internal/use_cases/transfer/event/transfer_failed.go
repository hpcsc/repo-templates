package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferFailed struct {
	TransferID uuid.UUID `json:"transferId"`
	Reason     string    `json:"reason"`
}

func (TransferFailed) EventType() string {
	return domainEvent.Namespace + ".transfer-failed"
}
