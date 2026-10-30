package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type SanctionsScreeningFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Reason    string    `json:"reason"`
}

func (SanctionsScreeningFailed) EventType() string {
	return domainEvent.Namespace + ".sanctions-screening-failed"
}
