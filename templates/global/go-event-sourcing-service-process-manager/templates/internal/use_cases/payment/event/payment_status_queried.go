package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentStatusQueried struct {
	PaymentID          uuid.UUID `json:"paymentId"`
	Attempt            int       `json:"attempt"`
	SettlementDeadline time.Time `json:"settlementDeadline"`
}

func (PaymentStatusQueried) EventType() string {
	return domainEvent.Namespace + ".payment-status-queried"
}
