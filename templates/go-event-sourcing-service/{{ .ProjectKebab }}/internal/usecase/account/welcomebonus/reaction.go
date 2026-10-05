package welcomebonus

import (
	"context"
	"fmt"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/dispatch"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
)

const Amount int64 = 1000

var _ reaction.Interface = (*Reaction)(nil)

func NewReaction(bus *command.Bus) *Reaction {
	return &Reaction{
		bus: bus,
	}
}

type Reaction struct {
	bus *command.Bus
}

func (r *Reaction) Name() string {
	return "welcome-bonus"
}

func (r *Reaction) EventTypes() []string {
	return []string{event.AccountOpened{}.EventType()}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	opened, ok := evt.Payload.(*event.AccountOpened)
	if !ok {
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}

	return r.bus.Dispatch(ctx, domain.Command[accountCommand.GrantWelcomeBonus]{
		ID: dispatch.CommandIDFor(r.Name(), evt),
		Payload: accountCommand.GrantWelcomeBonus{
			AccountID: opened.AccountID,
			Amount:    Amount,
		},
	})
}
