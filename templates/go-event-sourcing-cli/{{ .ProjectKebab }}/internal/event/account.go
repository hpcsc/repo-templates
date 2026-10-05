package event

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type AccountOpened struct {
	AccountID string    `json:"account_id"`
	Owner     string    `json:"owner"`
	OpenedAt  time.Time `json:"opened_at"`
}

var _ es.Event = AccountOpened{}

func (AccountOpened) Kind() string { return "account.opened" }

type AccountCredited struct {
	AccountID  string    `json:"account_id"`
	Amount     int64     `json:"amount"`
	Reference  string    `json:"reference"`
	CreditedAt time.Time `json:"credited_at"`
}

var _ es.Event = AccountCredited{}

func (AccountCredited) Kind() string { return "account.credited" }

func init() {
	register[AccountOpened]()
	register[AccountCredited]()
}
