package account

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
)

var _ stream.Projection = (*State)(nil)

func StreamID(accountID string) string {
	return "account-" + accountID
}

type State struct {
	Open     bool
	Balance  int64
	credited map[string]bool
}

func (s *State) Apply(record es.Record) {
	switch e := record.Event.(type) {
	case event.AccountOpened:
		s.Open = true
	case event.AccountCredited:
		s.Balance += e.Amount
		if s.credited == nil {
			s.credited = map[string]bool{}
		}
		s.credited[e.Reference] = true
	}
}

func (s *State) Credited(reference string) bool {
	return s.credited[reference]
}
