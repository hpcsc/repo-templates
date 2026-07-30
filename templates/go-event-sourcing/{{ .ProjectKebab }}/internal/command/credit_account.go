package command

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type CreditAccount struct {
	AccountID  string
	Amount     int64
	Reference  string
	CreditedAt time.Time
}

func (c CreditAccount) AggregateID() string { return c.AccountID }

var _ es.Command = CreditAccount{}
