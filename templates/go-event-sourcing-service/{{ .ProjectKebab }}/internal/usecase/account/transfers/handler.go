package transfers

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
)

func Register(bus *command.Bus, accounts *account.Repository) {
	bus.Register[accountCommand.DebitForTransfer](debitHandler{accounts: accounts})
	bus.Register[accountCommand.CreditForTransfer](creditHandler{accounts: accounts})
	bus.Register[accountCommand.RefundTransfer](refundHandler{accounts: accounts})
}

type debitHandler struct {
	accounts *account.Repository
}

func (h debitHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.DebitForTransfer]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.DebitForTransfer(cmd.Payload.TransferID, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.ID)
}

type creditHandler struct {
	accounts *account.Repository
}

func (h creditHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.CreditForTransfer]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.CreditForTransfer(cmd.Payload.TransferID, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.ID)
}

type refundHandler struct {
	accounts *account.Repository
}

func (h refundHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.RefundTransfer]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.RefundTransfer(cmd.Payload.TransferID, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.ID)
}
