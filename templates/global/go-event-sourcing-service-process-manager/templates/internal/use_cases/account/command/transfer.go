package command

import "github.com/google/uuid"

type DebitForTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}

type CreditForTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}

type RefundTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}
