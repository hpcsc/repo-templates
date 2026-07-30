package command

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type OpenAccount struct {
	AccountID string
	Owner     string
	OpenedAt  time.Time
}

func (c OpenAccount) AggregateID() string { return c.AccountID }

var _ es.Command = OpenAccount{}
