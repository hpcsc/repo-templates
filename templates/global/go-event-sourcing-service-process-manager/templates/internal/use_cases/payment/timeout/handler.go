package timeout

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
)

func Register(bus *command.Bus, payments *payment.Repository, timeouts payment.Timeouts, now func() time.Time) {
	h := handlers{payments: payments, timeouts: timeouts, now: now}
	bus.Register[paymentCommand.TimeOutChecks](checksHandler{h})
	bus.Register[paymentCommand.TimeOutSubmission](submissionHandler{h})
	bus.Register[paymentCommand.TimeOutSettlement](settlementHandler{h})
}

type handlers struct {
	payments *payment.Repository
	timeouts payment.Timeouts
	now      func() time.Time
}

func (h handlers) change(ctx context.Context, paymentID uuid.UUID, cause domain.Cause, decide func(*payment.Payment) error) error {
	p, err := h.payments.Load(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := decide(p); err != nil {
		return err
	}
	return h.payments.Save(ctx, p, cause)
}

type checksHandler struct {
	handlers
}

func (h checksHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.TimeOutChecks]) error {
	return h.change(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.TimeOutChecks(cmd.Payload.Attempt, h.now().Add(h.timeouts.Checks))
	})
}

type submissionHandler struct {
	handlers
}

func (h submissionHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.TimeOutSubmission]) error {
	return h.change(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.TimeOutSubmission(cmd.Payload.Attempt, h.now().Add(h.timeouts.Submission))
	})
}

type settlementHandler struct {
	handlers
}

func (h settlementHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.TimeOutSettlement]) error {
	return h.change(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.TimeOutSettlement(cmd.Payload.Attempt, h.now().Add(h.timeouts.Settlement))
	})
}
