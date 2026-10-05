package closeaccount

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
)

func Register(bus *command.Bus, accounts *account.Repository) {
	bus.Register[accountCommand.Close](handler{accounts: accounts})
}

type handler struct {
	accounts *account.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[accountCommand.Close]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.AccountID)
	if err != nil {
		return err
	}
	if err := a.Close(); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.ID)
}
