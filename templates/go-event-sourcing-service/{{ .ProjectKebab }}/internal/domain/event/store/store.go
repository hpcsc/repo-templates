package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type OfEvents struct {
	stream event.Stream
}

func New(stream event.Stream) *OfEvents {
	return &OfEvents{
		stream: stream,
	}
}

func (s *OfEvents) Load[T any, A interface {
	*T
	domain.Aggregate
}](ctx context.Context, streamID string) (A, error) {
	history, err := s.stream.EventsForStream(ctx, streamID)
	if err != nil {
		return nil, err
	}

	aggregate := A(new(T))
	for _, e := range history {
		aggregate.Apply(e)
	}
	aggregate.MarkCommitted(uint64(len(history)))

	return aggregate, nil
}

func (s *OfEvents) Save(ctx context.Context, streamID string, aggregate domain.Aggregate, causeID uuid.UUID) error {
	events := aggregate.Recorded()
	for i := range events {
		events[i] = events[i].CorrelateWithCommandID(causeID)
	}

	if err := s.stream.Save(ctx, streamID, events, aggregate.Version()); err != nil {
		return err
	}

	aggregate.MarkCommitted(aggregate.Version() + uint64(len(events)))
	return nil
}
