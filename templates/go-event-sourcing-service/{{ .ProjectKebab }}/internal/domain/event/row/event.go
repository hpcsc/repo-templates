package row

import (
	"fmt"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/jackc/pgx/v5"
)

const Columns = "id::text, stream_id, version, transaction_id, sequence, type, schema_version, data, metadata"

type Event struct {
	ID            string
	StreamID      string
	Version       uint64
	Position      domain.Position
	Type          string
	SchemaVersion int
	Data          []byte
	Metadata      []byte
}

func Scan(rows pgx.Rows) ([]Event, error) {
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Event, error) {
		var e Event
		err := r.Scan(&e.ID, &e.StreamID, &e.Version, &e.Position.TransactionID, &e.Position.Sequence, &e.Type, &e.SchemaVersion, &e.Data, &e.Metadata)
		return e, err
	})
}

func (e Event) ToDomain(reg *registry.OfEvents) (domain.Event, error) {
	payload, metadata, err := reg.ParseRawData(e.Type, e.SchemaVersion, e.Data, e.Metadata)
	if err != nil {
		return domain.Event{}, fmt.Errorf("failed to parse event %s at version %d of stream %s: %w", e.ID, e.Version, e.StreamID, err)
	}

	return domain.Event{
		ID:       e.ID,
		StreamID: e.StreamID,
		Version:  e.Version,
		Position: e.Position,
		Payload:  payload,
		Metadata: metadata,
	}, nil
}
