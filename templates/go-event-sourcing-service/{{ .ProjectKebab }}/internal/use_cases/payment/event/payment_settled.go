package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentSettled struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
}

func (PaymentSettled) EventType() string {
	return domainEvent.Namespace + ".payment-settled"
}
