package command

import "github.com/google/uuid"

type RecordScreeningResult struct {
	PaymentID uuid.UUID
	Passed    bool
	Reason    string
}
