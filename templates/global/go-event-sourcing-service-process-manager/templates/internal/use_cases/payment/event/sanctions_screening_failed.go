package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type SanctionsScreeningFailed struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Reason    string    `json:"reason"`
}

func (SanctionsScreeningFailed) EventType() string {
	return domainEvent.Namespace + ".sanctions-screening-failed"
}
