package command

import "github.com/google/uuid"

type TimeOutSettlement struct {
	PaymentID uuid.UUID
	Attempt   int
}
