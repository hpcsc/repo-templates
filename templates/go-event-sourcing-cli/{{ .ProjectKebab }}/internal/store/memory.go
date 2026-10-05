package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
)

var _ stream.Store = (*Memory)(nil)

type Memory struct {
	mutex   sync.Mutex
	streams map[string][]es.Record
}

func NewMemory() *Memory {
	return &Memory{streams: map[string][]es.Record{}}
}

func (m *Memory) Load(_ context.Context, streamID string) ([]es.Record, int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	records := append([]es.Record(nil), m.streams[streamID]...)
	return records, len(records), nil
}

func (m *Memory) Append(_ context.Context, streamID string, expectedSeq int, events ...es.Event) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	current := len(m.streams[streamID])
	if current != expectedSeq {
		return fmt.Errorf("%w: stream %q is at seq %d, not %d", stream.ErrVersionConflict, streamID, current, expectedSeq)
	}

	at := time.Now().UTC()
	for i, e := range events {
		m.streams[streamID] = append(m.streams[streamID], es.Record{StreamID: streamID, Seq: expectedSeq + 1 + i, At: at, Event: e})
	}
	return nil
}
