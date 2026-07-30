package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

// NewMemory returns a store that keeps streams in memory. It exists so unit
// tests can exercise aggregates and use cases without a database; anything
// that needs to survive the process uses Postgres.
//
// It round-trips events through the codec exactly as Postgres does, so a type
// that was never registered fails here too rather than only in production.
func NewMemory() es.Store {
	return &memory{streams: make(map[string][]stored)}
}

type stored struct {
	name          string
	schemaVersion int
	data          []byte
}

type memory struct {
	mu      sync.Mutex
	streams map[string][]stored
}

func (s *memory) Load(_ context.Context, streamID string) ([]es.Event, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := s.streams[streamID]
	history := make([]es.Event, 0, len(rows))
	for _, row := range rows {
		event, err := es.Decode(row.name, row.schemaVersion, row.data)
		if err != nil {
			return nil, 0, fmt.Errorf("stream %q: %w", streamID, err)
		}
		history = append(history, event)
	}

	return history, len(rows), nil
}

func (s *memory) Append(_ context.Context, streamID string, expectedVersion int, events []es.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.streams[streamID]) != expectedVersion {
		return fmt.Errorf("appending to stream %q: %w", streamID, es.ErrVersionConflict)
	}

	rows := make([]stored, 0, len(events))
	for _, event := range events {
		name, schemaVersion, data, err := es.Encode(event)
		if err != nil {
			return fmt.Errorf("stream %q: %w", streamID, err)
		}
		rows = append(rows, stored{name: name, schemaVersion: schemaVersion, data: data})
	}

	s.streams[streamID] = append(s.streams[streamID], rows...)
	return nil
}

var _ es.Store = (*memory)(nil)
