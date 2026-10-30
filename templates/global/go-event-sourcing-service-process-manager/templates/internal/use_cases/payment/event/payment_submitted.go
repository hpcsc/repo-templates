package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentSubmitted struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	NetworkReference   string    `json:"networkReference"`
	SettlementDeadline time.Time `json:"settlementDeadline"`
}

func (PaymentSubmitted) EventType() string {
	return domainEvent.Namespace + ".payment-submitted"
}
