package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type HoldRejected struct {
	AccountID uuid.UUID `json:"accountId"`
	PaymentID uuid.UUID `json:"paymentId"`
	Amount    int64     `json:"amount"`
	Reason    string    `json:"reason"`
}

func (HoldRejected) EventType() string {
	return domainEvent.Namespace + ".hold-rejected"
}
