package account

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.AccountOpened]()
	reg.Register[event.MoneyDeposited]()
	reg.Register[event.MoneyWithdrawn]()
	reg.Register[event.WelcomeBonusGranted]()
	reg.Register[event.AccountClosed]()
{{- if .Scaffold.ProcessManager }}
	reg.Register[event.TransferDebited]()
	reg.Register[event.TransferDebitRejected]()
	reg.Register[event.TransferCredited]()
	reg.Register[event.TransferCreditRejected]()
	reg.Register[event.TransferRefunded]()
{{- end }}
}
