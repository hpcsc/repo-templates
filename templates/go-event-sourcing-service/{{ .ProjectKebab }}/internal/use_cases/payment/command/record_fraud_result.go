package command

import "github.com/google/uuid"

type RecordFraudResult struct {
	PaymentID uuid.UUID
	Passed    bool
	Reason    string
}
