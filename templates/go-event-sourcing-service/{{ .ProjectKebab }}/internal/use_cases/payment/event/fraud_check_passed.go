package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type FraudCheckPassed struct {
	PaymentID uuid.UUID `json:"paymentId"`
}

func (FraudCheckPassed) EventType() string {
	return domainEvent.Namespace + ".fraud-check-passed"
}
