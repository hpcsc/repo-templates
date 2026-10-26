package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferDebited struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferDebited) EventType() string {
	return domainEvent.Namespace + ".transfer-debited"
}
