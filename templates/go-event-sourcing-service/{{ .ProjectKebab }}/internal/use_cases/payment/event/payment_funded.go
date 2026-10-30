package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentFunded struct {
	PaymentID      uuid.UUID `json:"paymentId"`
	ChecksDeadline time.Time `json:"checksDeadline"`
}

func (PaymentFunded) EventType() string {
	return domainEvent.Namespace + ".payment-funded"
}
