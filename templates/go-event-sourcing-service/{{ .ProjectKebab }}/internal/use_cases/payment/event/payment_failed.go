package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	AccountID uuid.UUID `json:"accountId"`
	Reason    string    `json:"reason"`
}

func (PaymentFailed) EventType() string {
	return domainEvent.Namespace + ".payment-failed"
}
