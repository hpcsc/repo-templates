package usecase

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/closeaccount"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/deposit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/open"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/transfers"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/welcomebonus"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/withdraw"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/start"
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
