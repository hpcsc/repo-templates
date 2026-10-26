package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type MoneyWithdrawn struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (MoneyWithdrawn) EventType() string {
	return domainEvent.Namespace + ".money-withdrawn"
}
