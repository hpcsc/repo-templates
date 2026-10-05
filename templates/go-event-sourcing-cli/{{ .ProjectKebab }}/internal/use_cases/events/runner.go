package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
)

type Line struct {
	Seq  int             `json:"seq"`
	At   time.Time       `json:"at"`
	Kind string          `json:"kind"`
	V    int             `json:"v"`
	Data json.RawMessage `json:"data"`
}

type Runner struct {
	store stream.Store
}

func New(store stream.Store) Runner {
	return Runner{store: store}
}

func (r Runner) Run(ctx context.Context, streamID string) ([]Line, error) {
	records, _, err := r.store.Load(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("read stream %q: %w", streamID, err)
	}

	lines := make([]Line, 0, len(records))
	for _, record := range records {
		kind, schemaVersion, data, err := event.Encode(record.Event)
		if err != nil {
			return nil, err
		}
		lines = append(lines, Line{Seq: record.Seq, At: record.At, Kind: kind, V: schemaVersion, Data: data})
	}
	return lines, nil
}
