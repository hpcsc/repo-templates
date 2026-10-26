package command

import "github.com/google/uuid"

type Deposit struct {
	AccountID uuid.UUID
	Amount    int64
}
