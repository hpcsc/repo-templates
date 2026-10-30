package command

import "github.com/google/uuid"

type CaptureHold struct {
	AccountID uuid.UUID
	PaymentID uuid.UUID
}
