package record

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
	r := recorder{payments: payments}
	bus.Register[paymentCommand.RecordFunding](fundingHandler{r, timeouts, now})
	bus.Register[paymentCommand.RecordFraudResult](fraudHandler{r, timeouts, now})
	bus.Register[paymentCommand.RecordScreeningResult](screeningHandler{r, timeouts, now})
	bus.Register[paymentCommand.RecordSubmission](submissionHandler{r, timeouts, now})
	bus.Register[paymentCommand.RecordSettlement](settlementHandler{r})
	bus.Register[paymentCommand.RecordReturn](returnHandler{r})
}

type recorder struct {
	payments *payment.Repository
}

func (r recorder) record(ctx context.Context, paymentID uuid.UUID, cause domain.Cause, decide func(*payment.Payment) error) error {
	p, err := r.payments.Load(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := decide(p); err != nil {
		return err
	}
	return r.payments.Save(ctx, p, cause)
}

type fundingHandler struct {
	recorder
	timeouts payment.Timeouts
	now      func() time.Time
}

func (h fundingHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordFunding]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordFunding(cmd.Payload.Held, cmd.Payload.Reason, h.now().Add(h.timeouts.Checks))
	})
}

type fraudHandler struct {
	recorder
	timeouts payment.Timeouts
	now      func() time.Time
}

func (h fraudHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordFraudResult]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordFraudResult(cmd.Payload.Passed, cmd.Payload.Reason, h.now().Add(h.timeouts.Submission))
	})
}

type screeningHandler struct {
	recorder
	timeouts payment.Timeouts
	now      func() time.Time
}

func (h screeningHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordScreeningResult]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordScreeningResult(cmd.Payload.Passed, cmd.Payload.Reason, h.now().Add(h.timeouts.Submission))
	})
}

type submissionHandler struct {
	recorder
	timeouts payment.Timeouts
	now      func() time.Time
}

func (h submissionHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordSubmission]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordSubmission(cmd.Payload.NetworkReference, h.now().Add(h.timeouts.Settlement))
	})
}

type settlementHandler struct {
	recorder
}

func (h settlementHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordSettlement]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordSettlement()
	})
}

type returnHandler struct {
	recorder
}

func (h returnHandler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.RecordReturn]) error {
	return h.record(ctx, cmd.Payload.PaymentID, cmd.Cause(), func(p *payment.Payment) error {
		return p.RecordReturn(cmd.Payload.Reason)
	})
}
