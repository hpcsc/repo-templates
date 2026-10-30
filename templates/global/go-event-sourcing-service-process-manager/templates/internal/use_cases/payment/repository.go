package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/store"
	paymentEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
)

func NewRepository(stream event.Stream) *Repository {
	return &Repository{
		stream: stream,
		events: store.New(stream),
	}
}

type Repository struct {
	stream event.Stream
	events *store.OfEvents
}

func (r *Repository) Load(ctx context.Context, id uuid.UUID) (*Payment, error) {
	return r.events.Load[Payment](ctx, StreamID(id))
}

func (r *Repository) Save(ctx context.Context, p *Payment, cause domain.Cause) error {
	return r.events.Save(ctx, StreamID(p.ID()), p, cause)
}

// Requested and Submitted read the stream, because the aggregate keeps only what its rules read.
func (r *Repository) Requested(ctx context.Context, id uuid.UUID) (*paymentEvent.PaymentRequested, error) {
	return firstOf[*paymentEvent.PaymentRequested](ctx, r.stream, id, ErrNotRequested)
}

func (r *Repository) Submitted(ctx context.Context, id uuid.UUID) (*paymentEvent.PaymentSubmitted, error) {
	return firstOf[*paymentEvent.PaymentSubmitted](ctx, r.stream, id, ErrNotSubmitted)
}

func firstOf[P domain.EventPayload](ctx context.Context, stream event.Stream, id uuid.UUID, missing error) (P, error) {
	var none P
	events, err := stream.EventsForStream(ctx, StreamID(id))
	if err != nil {
		return none, err
	}
	for _, e := range events {
		if payload, ok := e.Payload.(P); ok {
			return payload, nil
		}
	}
	return none, fmt.Errorf("%w: %s", missing, id)
}
