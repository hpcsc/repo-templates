package command

import "github.com/google/uuid"

type DebitForTransfer struct {
	AccountID  uuid.UUID
	TransferID uuid.UUID
	Amount     int64
}
