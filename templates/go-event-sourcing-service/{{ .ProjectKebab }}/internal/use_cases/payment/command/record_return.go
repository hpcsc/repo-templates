package command

import "github.com/google/uuid"

type RecordReturn struct {
	PaymentID uuid.UUID
	Reason    string
}
