package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type FraudCheckPassed struct {
	PaymentID uuid.UUID `json:"paymentId"`
}

func (FraudCheckPassed) EventType() string {
	return domainEvent.Namespace + ".fraud-check-passed"
}
