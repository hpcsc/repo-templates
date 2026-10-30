package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/store"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
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

// Requested reads the request from the stream, because the aggregate keeps only what its rules read.
func (r *Repository) Requested(ctx context.Context, id uuid.UUID) (*paymentEvent.PaymentRequested, error) {
	events, err := r.stream.EventsForStream(ctx, StreamID(id))
	if err != nil {
		return nil, err
	}
	if len(events) > 0 {
		if requested, ok := events[0].Payload.(*paymentEvent.PaymentRequested); ok {
			return requested, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotRequested, id)
}
