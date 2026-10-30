package command

import "github.com/google/uuid"

type ReleaseHold struct {
	AccountID uuid.UUID
	PaymentID uuid.UUID
}
