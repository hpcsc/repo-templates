package use_cases

import (
{{- if .Scaffold.ProcessManager }}
	"time"
{{ end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/closeaccount"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/deposit"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/holds"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/welcomebonus"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/withdraw"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/record"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/request"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/timeout"
{{- end }}
)

func RegisterEvents(reg *registry.OfEvents) {
	account.RegisterEvents(reg)
{{- if .Scaffold.ProcessManager }}
	payment.RegisterEvents(reg)
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
	holds.Register(bus, accounts)

	payments := payment.NewRepository(stream)
	request.Register(bus, payments, accounts)
	record.Register(bus, payments, payment.DefaultTimeouts, time.Now)
	timeout.Register(bus, payments, payment.DefaultTimeouts, time.Now)
{{- end }}
}
