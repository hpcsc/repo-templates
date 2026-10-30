package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/store"
)

func NewRepository(stream event.Stream) *Repository {
	return &Repository{
		events: store.New(stream),
	}
}

type Repository struct {
	events *store.OfEvents
}

func (r *Repository) Load(ctx context.Context, id uuid.UUID) (*Payment, error) {
	return r.events.Load[Payment](ctx, StreamID(id))
}

func (r *Repository) Save(ctx context.Context, p *Payment, cause domain.Cause) error {
	return r.events.Save(ctx, StreamID(p.ID()), p, cause)
}
