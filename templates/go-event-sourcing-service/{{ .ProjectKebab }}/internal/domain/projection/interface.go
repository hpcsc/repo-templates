package projection

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/jackc/pgx/v5"
)

type Interface interface {
	Handle(ctx context.Context, tx pgx.Tx, event *domain.Event) error
	EventTypes() []string
	Name() string
}

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
