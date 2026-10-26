package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type WelcomeBonusGranted struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (WelcomeBonusGranted) EventType() string {
	return domainEvent.Namespace + ".welcome-bonus-granted"
}
