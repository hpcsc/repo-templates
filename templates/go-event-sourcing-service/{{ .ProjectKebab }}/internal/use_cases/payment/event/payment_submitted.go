package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentSubmitted struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	NetworkReference   string    `json:"networkReference"`
	SettlementDeadline time.Time `json:"settlementDeadline"`
}

func (PaymentSubmitted) EventType() string {
	return domainEvent.Namespace + ".payment-submitted"
}
