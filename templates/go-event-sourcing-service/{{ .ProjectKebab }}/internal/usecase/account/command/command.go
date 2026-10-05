package command

import "github.com/google/uuid"

type Open struct {
	AccountID uuid.UUID
	Owner     string
}

type Deposit struct {
	AccountID uuid.UUID
	Amount    int64
}

type Withdraw struct {
	AccountID uuid.UUID
	Amount    int64
}

type GrantWelcomeBonus struct {
	AccountID uuid.UUID
	Amount    int64
}

type Close struct {
	AccountID uuid.UUID
}
