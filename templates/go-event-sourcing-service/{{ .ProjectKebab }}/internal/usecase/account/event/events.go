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

type MoneyDeposited struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (MoneyDeposited) EventType() string {
	return domainEvent.Namespace + ".money-deposited"
}

type MoneyWithdrawn struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (MoneyWithdrawn) EventType() string {
	return domainEvent.Namespace + ".money-withdrawn"
}

type WelcomeBonusGranted struct {
	AccountID uuid.UUID `json:"accountId"`
	Amount    int64     `json:"amount"`
}

func (WelcomeBonusGranted) EventType() string {
	return domainEvent.Namespace + ".welcome-bonus-granted"
}

type AccountClosed struct {
	AccountID uuid.UUID `json:"accountId"`
}

func (AccountClosed) EventType() string {
	return domainEvent.Namespace + ".account-closed"
}
