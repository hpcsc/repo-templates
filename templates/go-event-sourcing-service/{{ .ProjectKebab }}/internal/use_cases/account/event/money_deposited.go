package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type MoneyDeposited struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (MoneyDeposited) EventType() string {
	return domainEvent.Namespace + ".money-deposited"
}
