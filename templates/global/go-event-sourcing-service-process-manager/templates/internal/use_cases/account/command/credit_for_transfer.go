package command

import "github.com/google/uuid"

type CreditForTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}
