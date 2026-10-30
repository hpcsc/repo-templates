package funding

import (
	"context"
	"fmt"

	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/reaction"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/reaction/dispatch"
	accountCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account/command"
	accountEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account/event"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
)

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
	return "payment-funding"
}

func (r *Reaction) EventTypes() []string {
	return []string{
		paymentEvent.PaymentRequested{}.EventType(),
		accountEvent.HoldPlaced{}.EventType(),
		accountEvent.HoldRejected{}.EventType(),
	}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *paymentEvent.PaymentRequested:
		return r.bus.Dispatch(ctx, dispatch.CommandFor(r.Name(), evt, accountCommand.PlaceHold{
			AccountID: payload.AccountID,
			PaymentID: payload.PaymentID,
			Amount:    payload.Amount,
		}))
	case *accountEvent.HoldPlaced:
		return r.bus.Dispatch(ctx, dispatch.CommandFor(r.Name(), evt, paymentCommand.RecordFunding{
			PaymentID: payload.PaymentID,
			Held:      true,
		}))
	case *accountEvent.HoldRejected:
		return r.bus.Dispatch(ctx, dispatch.CommandFor(r.Name(), evt, paymentCommand.RecordFunding{
			PaymentID: payload.PaymentID,
			Held:      false,
			Reason:    payload.Reason,
		}))
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}
