package es

import (
	"context"
	"errors"
)

// ErrVersionConflict reports that a stream moved on between being loaded and
// being appended to. A caller that reloads and re-decides resolves it.
var ErrVersionConflict = errors.New("es: stream version conflict")

type Store interface {
	Load(ctx context.Context, streamID string) (events []Event, version int, err error)
	Append(ctx context.Context, streamID string, expectedVersion int, events []Event) error
}
