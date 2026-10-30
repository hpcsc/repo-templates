package command

import "github.com/google/uuid"

type PlaceHold struct {
	AccountID uuid.UUID
	PaymentID uuid.UUID
	Amount    int64
}
