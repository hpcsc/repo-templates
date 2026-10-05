package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
)

type TransferStarted struct {
	TransferID uuid.UUID `json:"transferId"`
	From       uuid.UUID `json:"from"`
	To         uuid.UUID `json:"to"`
	Amount     int64     `json:"amount"`
}

func (TransferStarted) EventType() string {
	return domainEvent.Namespace + ".transfer-started"
}

type TransferDebitRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferDebitRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-debit-requested"
}

type TransferCreditRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferCreditRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-credit-requested"
}

type TransferRefundRequested struct {
	TransferID uuid.UUID `json:"transferId"`
	AccountID  uuid.UUID `json:"accountId"`
	Amount     int64     `json:"amount"`
}

func (TransferRefundRequested) EventType() string {
	return domainEvent.Namespace + ".transfer-refund-requested"
}

type TransferCompleted struct {
	TransferID uuid.UUID `json:"transferId"`
}

func (TransferCompleted) EventType() string {
	return domainEvent.Namespace + ".transfer-completed"
}

type TransferFailed struct {
	TransferID uuid.UUID `json:"transferId"`
	Reason     string    `json:"reason"`
}

func (TransferFailed) EventType() string {
	return domainEvent.Namespace + ".transfer-failed"
}
