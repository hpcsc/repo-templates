package command

import "github.com/google/uuid"

type Withdraw struct {
	AccountID uuid.UUID
	Amount    int64
}
