package stream

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/jackc/pgx/v5"
)

var _ event.Stream = (*memory)(nil)

type memory struct {
	streamsByID map[string][]domain.Event
	mutex       sync.RWMutex
}

func NewMemory() event.Stream {
	return &memory{
		streamsByID: make(map[string][]domain.Event),
	}
}

func (s *memory) Save(_ context.Context, streamID string, events []domain.Event, expectedVersion uint64) error {
	if len(events) == 0 {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	current := uint64(len(s.streamsByID[streamID]))
	if current != expectedVersion {
		return fmt.Errorf("%w: stream %s is at version %d, not %d", event.ErrConcurrencyConflict, streamID, current, expectedVersion)
	}

	if commandID := events[0].CommandID; commandID != uuid.Nil && s.hasCommand(commandID) {
		return fmt.Errorf("%w: command %s", event.ErrDuplicateCommand, commandID)
	}

	for i, evt := range events {
		if evt.ID == "" {
			evt.ID = uuid.NewString()
		}
		evt.StreamID = streamID
		evt.Version = expectedVersion + uint64(i) + 1
		s.streamsByID[streamID] = append(s.streamsByID[streamID], evt)
	}

	return nil
}

func (s *memory) SaveInTx(ctx context.Context, _ pgx.Tx, streamID string, events []domain.Event, expectedVersion uint64) error {
	return s.Save(ctx, streamID, events, expectedVersion)
}

func (s *memory) HasCommand(_ context.Context, commandID uuid.UUID) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.hasCommand(commandID), nil
}

func (s *memory) hasCommand(commandID uuid.UUID) bool {
	for _, events := range s.streamsByID {
		for _, evt := range events {
			if evt.CommandID == commandID {
				return true
			}
		}
	}
	return false
}

func (s *memory) EventsForStream(_ context.Context, streamID string) ([]domain.Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return append([]domain.Event(nil), s.streamsByID[streamID]...), nil
}
