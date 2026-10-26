package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type AccountClosed struct {
	AccountID uuid.UUID `json:"accountId"`
}

func (AccountClosed) EventType() string {
	return domainEvent.Namespace + ".account-closed"
}
