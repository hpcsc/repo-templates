package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentRefused struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
	Reason    string    `json:"reason"`
}

func (PaymentRefused) EventType() string {
	return domainEvent.Namespace + ".payment-refused"
}
