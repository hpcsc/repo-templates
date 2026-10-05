package start

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer"
	transferCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/command"
)

func Register(bus *command.Bus, transfers *transfer.Repository, accounts *account.Repository) {
	bus.Register[transferCommand.Start](handler{transfers: transfers, accounts: accounts})
}

type handler struct {
	transfers *transfer.Repository
	accounts  *account.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[transferCommand.Start]) error {
	for _, id := range []uuid.UUID{cmd.Payload.From, cmd.Payload.To} {
		if err := h.checkExists(ctx, id); err != nil {
			return err
		}
	}

	t, err := h.transfers.Load(ctx, cmd.Payload.TransferID)
	if err != nil {
		return err
	}
	if err := t.Start(cmd.Payload.TransferID, cmd.Payload.From, cmd.Payload.To, cmd.Payload.Amount); err != nil {
		return err
	}
	return h.transfers.Save(ctx, t, cmd.ID)
}

func (h handler) checkExists(ctx context.Context, accountID uuid.UUID) error {
	a, err := h.accounts.Load(ctx, accountID)
	if err != nil {
		return err
	}
	if a.ID() == uuid.Nil {
		return fmt.Errorf("%w: %s", transfer.ErrUnknownAccount, accountID)
	}
	return nil
}
