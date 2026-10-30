package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentChecksRetried struct {
	PaymentID      uuid.UUID `json:"paymentId"`
	Attempt        int       `json:"attempt"`
	ChecksDeadline time.Time `json:"checksDeadline"`
}

func (PaymentChecksRetried) EventType() string {
	return domainEvent.Namespace + ".payment-checks-retried"
}
