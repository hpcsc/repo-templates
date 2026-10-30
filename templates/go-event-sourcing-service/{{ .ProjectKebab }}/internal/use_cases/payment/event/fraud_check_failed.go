package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type FraudCheckFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Reason    string    `json:"reason"`
}

func (FraudCheckFailed) EventType() string {
	return domainEvent.Namespace + ".fraud-check-failed"
}
