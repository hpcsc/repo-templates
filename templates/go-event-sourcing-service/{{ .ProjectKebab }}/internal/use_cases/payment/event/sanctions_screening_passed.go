package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type SanctionsScreeningPassed struct {
	PaymentID uuid.UUID `json:"paymentId"`
}

func (SanctionsScreeningPassed) EventType() string {
	return domainEvent.Namespace + ".sanctions-screening-passed"
}
