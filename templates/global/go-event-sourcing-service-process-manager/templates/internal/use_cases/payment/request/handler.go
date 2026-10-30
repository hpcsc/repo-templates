package request

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
)

func Register(bus *command.Bus, payments *payment.Repository, accounts *account.Repository) {
	bus.Register[paymentCommand.Request](handler{payments: payments, accounts: accounts})
}

type handler struct {
	payments *payment.Repository
	accounts *account.Repository
}

// A hold on an account that does not exist records no event, and the payment then never ends.
func (h handler) Handle(ctx context.Context, cmd domain.Command[paymentCommand.Request]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if a.ID() == uuid.Nil {
		return fmt.Errorf("%w: %s", payment.ErrUnknownAccount, cmd.Payload.AccountID)
	}

	p, err := h.payments.Load(ctx, cmd.Payload.PaymentID)
	if err != nil {
		return err
	}
	if err := p.Request(cmd.Payload.PaymentID, cmd.Payload.AccountID, cmd.Payload.CreditorIBAN, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.payments.Save(ctx, p, cmd.Cause())
}
