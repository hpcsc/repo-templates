package holds

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/command"
)

func Register(bus *command.Bus, accounts *account.Repository) {
	bus.Register[accountCommand.PlaceHold](placeHandler{accounts: accounts})
	bus.Register[accountCommand.ReleaseHold](releaseHandler{accounts: accounts})
	bus.Register[accountCommand.CaptureHold](captureHandler{accounts: accounts})
}

type placeHandler struct {
	accounts *account.Repository
}

func (h placeHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.PlaceHold]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.PlaceHold(cmd.Payload.PaymentID, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.Cause())
}

type releaseHandler struct {
	accounts *account.Repository
}

func (h releaseHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.ReleaseHold]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.ReleaseHold(cmd.Payload.PaymentID); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.Cause())
}

type captureHandler struct {
	accounts *account.Repository
}

func (h captureHandler) Handle(ctx context.Context, cmd domain.Command[accountCommand.CaptureHold]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.CaptureHold(cmd.Payload.PaymentID); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.Cause())
}
