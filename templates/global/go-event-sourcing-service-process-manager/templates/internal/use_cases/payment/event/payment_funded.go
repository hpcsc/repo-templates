package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentFunded struct {
	PaymentID      uuid.UUID `json:"paymentId"`
	ChecksDeadline time.Time `json:"checksDeadline"`
}

func (PaymentFunded) EventType() string {
	return domainEvent.Namespace + ".payment-funded"
}
