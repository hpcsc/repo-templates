package use_cases

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/closeaccount"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/deposit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/transfers"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/welcomebonus"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/withdraw"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer/start"
{{- end }}
)

func RegisterEvents(reg *registry.OfEvents) {
	account.RegisterEvents(reg)
{{- if .Scaffold.ProcessManager }}
	transfer.RegisterEvents(reg)
{{- end }}
}

func RegisterHandlers(bus *command.Bus, stream event.Stream) {
	accounts := account.NewRepository(stream)

	open.Register(bus, accounts)
	deposit.Register(bus, accounts)
	withdraw.Register(bus, accounts)
	welcomebonus.Register(bus, accounts)
	closeaccount.Register(bus, accounts)
{{- if .Scaffold.ProcessManager }}
	transfers.Register(bus, accounts)
	start.Register(bus, transfer.NewRepository(stream), accounts)
{{- end }}
}
