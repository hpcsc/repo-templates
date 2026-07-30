package show

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
)

type accountProjection struct {
	report Report
}

func (p *accountProjection) apply(history []es.Event) {
	for _, e := range history {
		switch applied := e.(type) {
		case event.AccountOpened:
			p.report.AccountID = applied.AccountID
			p.report.Owner = applied.Owner
			p.report.OpenedAt = applied.OpenedAt

		case event.AccountCredited:
			p.report.Balance += applied.Amount
			p.report.Credits = append(p.report.Credits, Credit{
				Amount:     applied.Amount,
				Reference:  applied.Reference,
				CreditedAt: applied.CreditedAt,
			})
		}
	}
}
