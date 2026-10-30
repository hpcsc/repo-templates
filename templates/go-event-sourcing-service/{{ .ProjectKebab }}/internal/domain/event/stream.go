package event

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/jackc/pgx/v5"
)

var (
	ErrConcurrencyConflict = errors.New("the stream changed after it was read")
	ErrDuplicateCommand    = errors.New("the store already has the events of the command")
)

type Stream interface {
	Save(ctx context.Context, streamID string, events []domain.Event, expectedVersion uint64) error
	SaveInTx(ctx context.Context, tx pgx.Tx, streamID string, events []domain.Event, expectedVersion uint64) error
	EventsForStream(ctx context.Context, streamID string) ([]domain.Event, error)
	HasCommand(ctx context.Context, commandID uuid.UUID) (bool, error)
}
