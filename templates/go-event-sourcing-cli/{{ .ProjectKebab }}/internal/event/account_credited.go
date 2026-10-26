package event

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type AccountCredited struct {
	AccountID  string    `json:"account_id"`
	Amount     int64     `json:"amount"`
	Reference  string    `json:"reference"`
	CreditedAt time.Time `json:"credited_at"`
}

var _ es.Event = AccountCredited{}

func (AccountCredited) Kind() string { return "account.credited" }

func init() {
	register[AccountCredited]()
}
