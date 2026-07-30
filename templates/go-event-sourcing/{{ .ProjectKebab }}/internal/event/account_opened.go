package event

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type AccountOpened struct {
	AccountID string
	Owner     string
	OpenedAt  time.Time
}

var _ es.Event = AccountOpened{}

// EventName is the name this event is stored under. Renaming the Go type is
// safe; changing this string is not, because old rows still carry the old one.
func (AccountOpened) EventName() string { return "AccountOpened" }

func init() {
	es.Register[AccountOpened]()
}
