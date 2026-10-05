package process

import (
	"context"
	"fmt"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/dispatch"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
)

var _ reaction.Interface = (*Dispatcher)(nil)

func NewDispatcher(bus *command.Bus) *Dispatcher {
	return &Dispatcher{
		bus: bus,
	}
}

type Dispatcher struct {
	bus *command.Bus
}

func (d *Dispatcher) Name() string {
	return "transfer-requests"
}

func (d *Dispatcher) EventTypes() []string {
	return []string{
		transferEvent.TransferDebitRequested{}.EventType(),
		transferEvent.TransferCreditRequested{}.EventType(),
		transferEvent.TransferRefundRequested{}.EventType(),
	}
}

func (d *Dispatcher) Handle(ctx context.Context, evt *domain.Event) error {
	id := dispatch.CommandIDFor(d.Name(), evt)

	switch e := evt.Payload.(type) {
	case *transferEvent.TransferDebitRequested:
		return d.bus.Dispatch(ctx, domain.Command[accountCommand.DebitForTransfer]{
			ID:      id,
			Payload: accountCommand.DebitForTransfer{AccountID: e.AccountID, TransferID: e.TransferID, Amount: e.Amount},
		})
	case *transferEvent.TransferCreditRequested:
		return d.bus.Dispatch(ctx, domain.Command[accountCommand.CreditForTransfer]{
			ID:      id,
			Payload: accountCommand.CreditForTransfer{AccountID: e.AccountID, TransferID: e.TransferID, Amount: e.Amount},
		})
	case *transferEvent.TransferRefundRequested:
		return d.bus.Dispatch(ctx, domain.Command[accountCommand.RefundTransfer]{
			ID:      id,
			Payload: accountCommand.RefundTransfer{AccountID: e.AccountID, TransferID: e.TransferID, Amount: e.Amount},
		})
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}
