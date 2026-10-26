package command

import "github.com/google/uuid"

type GrantWelcomeBonus struct {
	AccountID uuid.UUID
	Amount    int64
}
