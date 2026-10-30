package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type PaymentRequested struct {
	PaymentID    uuid.UUID `json:"paymentId"`
	AccountID    uuid.UUID `json:"accountId"`
	CreditorIBAN string    `json:"creditorIban"`
	Amount       int64     `json:"amount"`
}

func (PaymentRequested) EventType() string {
	return domainEvent.Namespace + ".payment-requested"
}
