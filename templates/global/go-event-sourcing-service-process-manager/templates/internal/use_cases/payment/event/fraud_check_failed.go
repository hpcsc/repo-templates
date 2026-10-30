package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type FraudCheckFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Reason    string    `json:"reason"`
}

func (FraudCheckFailed) EventType() string {
	return domainEvent.Namespace + ".fraud-check-failed"
}
