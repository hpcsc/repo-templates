package command

import "github.com/google/uuid"

type Start struct {
	TransferID uuid.UUID
	From       uuid.UUID
	To         uuid.UUID
	Amount     int64
}
