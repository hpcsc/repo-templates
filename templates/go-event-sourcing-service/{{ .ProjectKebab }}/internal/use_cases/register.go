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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/timeout"
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

type Option func(*options)

type options struct {
	paymentTimeouts payment.Timeouts
}

func WithPaymentTimeouts(timeouts payment.Timeouts) Option {
	return func(o *options) {
		o.paymentTimeouts = timeouts
	}
}

func RegisterHandlers(bus *command.Bus, stream event.Stream, opts ...Option) {
	o := options{paymentTimeouts: payment.DefaultTimeouts}
	for _, opt := range opts {
		opt(&o)
	}

	accounts := account.NewRepository(stream)

	open.Register(bus, accounts)
	deposit.Register(bus, accounts)
	withdraw.Register(bus, accounts)
	welcomebonus.Register(bus, accounts)
	closeaccount.Register(bus, accounts)
	holds.Register(bus, accounts)

	payments := payment.NewRepository(stream)
	request.Register(bus, payments, accounts)
	record.Register(bus, payments, o.paymentTimeouts, time.Now)
	timeout.Register(bus, payments, o.paymentTimeouts, time.Now)
{{- if .Scaffold.ProcessManager }}
	transfers.Register(bus, accounts)
	start.Register(bus, transfer.NewRepository(stream), accounts)
{{- end }}
}
