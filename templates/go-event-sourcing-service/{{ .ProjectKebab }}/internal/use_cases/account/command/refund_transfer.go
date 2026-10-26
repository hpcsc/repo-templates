package command

import "github.com/google/uuid"

type RefundTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}
