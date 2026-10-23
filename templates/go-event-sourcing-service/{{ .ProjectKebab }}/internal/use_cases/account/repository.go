package account

import (
	"context"

	"github.com/google/uuid"
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

func (r *Repository) Load(ctx context.Context, id uuid.UUID) (*Account, error) {
	return r.events.Load[Account](ctx, StreamID(id))
}

func (r *Repository) Save(ctx context.Context, a *Account, causeID uuid.UUID) error {
	return r.events.Save(ctx, StreamID(a.ID()), a, causeID)
}
