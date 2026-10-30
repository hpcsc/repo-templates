package command

import "github.com/google/uuid"

type TimeOutChecks struct {
	PaymentID uuid.UUID
	Attempt   int
}
