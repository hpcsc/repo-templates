package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferCredited struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferCredited) EventType() string {
	return domainEvent.Namespace + ".transfer-credited"
}
