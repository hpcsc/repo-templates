package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentCleared struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	SubmissionDeadline time.Time `json:"submissionDeadline"`
}

func (PaymentCleared) EventType() string {
	return domainEvent.Namespace + ".payment-cleared"
}
