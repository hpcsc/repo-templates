package command

import "github.com/google/uuid"

type RecordFunding struct {
	PaymentID uuid.UUID
	Held      bool
	Reason    string
}
