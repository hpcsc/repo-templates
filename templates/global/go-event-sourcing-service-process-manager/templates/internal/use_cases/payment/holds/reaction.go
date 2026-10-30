package holds

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/reaction"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/reaction/dispatch"
	accountCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/account/command"
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
	return "payment-holds"
}

func (r *Reaction) EventTypes() []string {
	return []string{
		paymentEvent.PaymentSettled{}.EventType(),
		paymentEvent.PaymentRefused{}.EventType(),
		paymentEvent.PaymentFailed{}.EventType(),
		paymentEvent.PaymentReturned{}.EventType(),
	}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *paymentEvent.PaymentSettled:
		return r.bus.Dispatch(ctx, dispatch.CommandFor(r.Name(), evt, accountCommand.CaptureHold{
			AccountID: payload.AccountID,
			PaymentID: payload.PaymentID,
		}))
	case *paymentEvent.PaymentRefused:
		return r.release(ctx, evt, payload.AccountID, payload.PaymentID)
	case *paymentEvent.PaymentFailed:
		return r.release(ctx, evt, payload.AccountID, payload.PaymentID)
	case *paymentEvent.PaymentReturned:
		return r.release(ctx, evt, payload.AccountID, payload.PaymentID)
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}

func (r *Reaction) release(ctx context.Context, evt *domain.Event, accountID uuid.UUID, paymentID uuid.UUID) error {
	return r.bus.Dispatch(ctx, dispatch.CommandFor(r.Name(), evt, accountCommand.ReleaseHold{
		AccountID: accountID,
		PaymentID: paymentID,
	}))
}
