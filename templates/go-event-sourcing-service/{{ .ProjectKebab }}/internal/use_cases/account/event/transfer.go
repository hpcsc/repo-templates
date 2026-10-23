package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type TransferDebited struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferDebited) EventType() string {
	return domainEvent.Namespace + ".transfer-debited"
}

type TransferDebitRejected struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"`
}

func (TransferDebitRejected) EventType() string {
	return domainEvent.Namespace + ".transfer-debit-rejected"
}

type TransferCredited struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferCredited) EventType() string {
	return domainEvent.Namespace + ".transfer-credited"
}

type TransferCreditRejected struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"`
}

func (TransferCreditRejected) EventType() string {
	return domainEvent.Namespace + ".transfer-credit-rejected"
}

type TransferRefunded struct {
	AccountID  uuid.UUID `json:"accountId"`
	TransferID uuid.UUID `json:"transferId"`
	Amount     int64     `json:"amount"`
}

func (TransferRefunded) EventType() string {
	return domainEvent.Namespace + ".transfer-refunded"
}
