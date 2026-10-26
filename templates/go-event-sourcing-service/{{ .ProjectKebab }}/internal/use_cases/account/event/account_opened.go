package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type AccountOpened struct {
	AccountID uuid.UUID `json:"accountId"`
	Owner     string    `json:"owner"`
}

func (AccountOpened) EventType() string {
	return domainEvent.Namespace + ".account-opened"
}
