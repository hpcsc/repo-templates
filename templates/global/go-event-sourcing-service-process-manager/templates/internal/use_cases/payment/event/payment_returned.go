package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentReturned struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
	Reason    string    `json:"reason"`
}

func (PaymentReturned) EventType() string {
	return domainEvent.Namespace + ".payment-returned"
}
