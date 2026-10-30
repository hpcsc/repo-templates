package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentSettled struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
}

func (PaymentSettled) EventType() string {
	return domainEvent.Namespace + ".payment-settled"
}
