package use_cases

import (
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/closeaccount"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/deposit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/holds"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/transfers"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/welcomebonus"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/withdraw"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/record"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/request"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer/start"
{{- end }}
)

func RegisterEvents(reg *registry.OfEvents) {
	account.RegisterEvents(reg)
	payment.RegisterEvents(reg)
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
	holds.Register(bus, accounts)

	payments := payment.NewRepository(stream)
	request.Register(bus, payments, accounts)
	record.Register(bus, payments, payment.DefaultTimeouts, time.Now)
{{- if .Scaffold.ProcessManager }}
	transfers.Register(bus, accounts)
	start.Register(bus, transfer.NewRepository(stream), accounts)
{{- end }}
}
