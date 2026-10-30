package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentStatusQueried struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	Attempt            int       `json:"attempt"`
	SettlementDeadline time.Time `json:"settlementDeadline"`
}

func (PaymentStatusQueried) EventType() string {
	return domainEvent.Namespace + ".payment-status-queried"
}
