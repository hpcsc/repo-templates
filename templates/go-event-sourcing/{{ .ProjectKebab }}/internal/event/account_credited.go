package event

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type AccountCredited struct {
	AccountID  string
	Amount     int64
	Reference  string
	CreditedAt time.Time
}

var _ es.Event = AccountCredited{}

func (AccountCredited) EventName() string { return "AccountCredited" }

func init() {
	es.Register[AccountCredited]()
}
