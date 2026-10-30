package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentEscalated struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Reason    string    `json:"reason"`
}

func (PaymentEscalated) EventType() string {
	return domainEvent.Namespace + ".payment-escalated"
}
