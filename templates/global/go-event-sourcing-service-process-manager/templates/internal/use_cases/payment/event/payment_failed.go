package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
	Reason    string    `json:"reason"`
}

func (PaymentFailed) EventType() string {
	return domainEvent.Namespace + ".payment-failed"
}
