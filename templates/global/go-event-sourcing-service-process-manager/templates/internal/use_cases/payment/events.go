package payment

import (
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/registry"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.PaymentRequested]()
	reg.Register[event.PaymentFunded]()
	reg.Register[event.PaymentFailed]()
	reg.Register[event.FraudCheckPassed]()
	reg.Register[event.FraudCheckFailed]()
	reg.Register[event.SanctionsScreeningPassed]()
	reg.Register[event.SanctionsScreeningFailed]()
	reg.Register[event.PaymentCleared]()
	reg.Register[event.PaymentRefused]()
	reg.Register[event.PaymentSubmitted]()
	reg.Register[event.PaymentSettled]()
	reg.Register[event.PaymentReturned]()
	reg.Register[event.PaymentChecksRetried]()
	reg.Register[event.PaymentSubmissionRetried]()
	reg.Register[event.PaymentStatusQueried]()
	reg.Register[event.PaymentEscalated]()
}
