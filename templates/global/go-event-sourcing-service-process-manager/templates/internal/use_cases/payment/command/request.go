package command

import "github.com/google/uuid"

type Request struct {
	PaymentID    uuid.UUID
	AccountID    uuid.UUID
	CreditorIBAN string
	Amount       int64
}
