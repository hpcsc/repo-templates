package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type PaymentSubmissionRetried struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	Attempt            int       `json:"attempt"`
	SubmissionDeadline time.Time `json:"submissionDeadline"`
}

func (PaymentSubmissionRetried) EventType() string {
	return domainEvent.Namespace + ".payment-submission-retried"
}
