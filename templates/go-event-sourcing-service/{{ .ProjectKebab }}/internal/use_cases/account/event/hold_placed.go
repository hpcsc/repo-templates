package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type HoldPlaced struct {
	AccountID uuid.UUID `json:"accountId"`
	PaymentID uuid.UUID `json:"paymentId"`
	Amount    int64     `json:"amount"`
}

func (HoldPlaced) EventType() string {
	return domainEvent.Namespace + ".hold-placed"
}
