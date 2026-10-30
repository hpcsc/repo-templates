package command

import "github.com/google/uuid"

type TimeOutSubmission struct {
	PaymentID uuid.UUID
	Attempt   int
}
