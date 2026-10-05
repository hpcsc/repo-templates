package transfer

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/store"
)

func NewRepository(stream event.Stream) *Repository {
	return &Repository{
		events: store.New(stream),
	}
}

type Repository struct {
	events *store.OfEvents
}

func (r *Repository) Load(ctx context.Context, id uuid.UUID) (*Transfer, error) {
	return r.events.Load[Transfer](ctx, StreamID(id))
}

func (r *Repository) Save(ctx context.Context, t *Transfer, causeID uuid.UUID) error {
	return r.events.Save(ctx, StreamID(t.ID()), t, causeID)
}
