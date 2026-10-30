package account

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.AccountOpened]()
	reg.Register[event.MoneyDeposited]()
	reg.Register[event.MoneyWithdrawn]()
	reg.Register[event.WelcomeBonusGranted]()
	reg.Register[event.AccountClosed]()
{{- if .Scaffold.ProcessManager }}
	reg.Register[event.HoldPlaced]()
	reg.Register[event.HoldRejected]()
	reg.Register[event.HoldReleased]()
	reg.Register[event.HoldCaptured]()
{{- end }}
}
