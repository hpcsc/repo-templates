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

func init() {
	register[AccountOpened]()
}
