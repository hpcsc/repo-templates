package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/row"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

var _ event.Stream = (*postgres)(nil)

func NewPostgres(pool *pgxpool.Pool, reg *registry.OfEvents) event.Stream {
	return &postgres{
		pool:     pool,
		registry: reg,
	}
}

type postgres struct {
	pool     *pgxpool.Pool
	registry *registry.OfEvents
}

func (s *postgres) Save(ctx context.Context, streamID string, events []domain.Event, expectedVersion uint64) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for stream %s: %w", streamID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.SaveInTx(ctx, tx, streamID, events, expectedVersion); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stream %s: %w", streamID, err)
	}
	return nil
}

func (s *postgres) SaveInTx(ctx context.Context, tx pgx.Tx, streamID string, events []domain.Event, expectedVersion uint64) error {
	if len(events) == 0 {
		return nil
	}

	// keep this read before the first write: the first write takes the transaction ID that orders the events
	var current uint64
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM events WHERE stream_id = $1", streamID).Scan(&current); err != nil {
		return fmt.Errorf("failed to read the version of stream %s: %w", streamID, err)
	}
	if current != expectedVersion {
		return fmt.Errorf("%w: stream %s is at version %d, not %d", event.ErrConcurrencyConflict, streamID, current, expectedVersion)
	}

	if commandID := events[0].CommandID; commandID != uuid.Nil {
		handled, err := hasCommand(ctx, tx, commandID)
		if err != nil {
			return err
		}
		if handled {
			return fmt.Errorf("%w: command %s", event.ErrDuplicateCommand, commandID)
		}
	}

	for i, evt := range events {
		if err := s.insert(ctx, tx, streamID, expectedVersion+uint64(i)+1, evt); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", event.NotifyChannel); err != nil {
		return fmt.Errorf("failed to notify about stream %s: %w", streamID, err)
	}
	return nil
}

func (s *postgres) insert(ctx context.Context, tx pgx.Tx, streamID string, version uint64, evt domain.Event) error {
	id := evt.ID
	if id == "" {
		id = uuid.NewString()
	}

	schemaVersion, err := s.registry.SchemaVersionOf(evt.Type())
	if err != nil {
		return fmt.Errorf("failed to append version %d to stream %s: %w", version, streamID, err)
	}

	data, err := json.Marshal(evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload of event %s: %w", evt.Type(), err)
	}

	var metadata []byte
	if evt.Metadata != nil {
		if metadata, err = json.Marshal(evt.Metadata); err != nil {
			return fmt.Errorf("failed to marshal metadata of event %s: %w", evt.Type(), err)
		}
	}

	var commandID any
	if evt.CommandID != uuid.Nil {
		commandID = evt.CommandID
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO events (id, stream_id, version, type, schema_version, data, metadata, command_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		id, streamID, version, evt.Type(), schemaVersion, data, metadata, commandID,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return fmt.Errorf("%w: stream %s already has version %d", event.ErrConcurrencyConflict, streamID, version)
	}
	if err != nil {
		return fmt.Errorf("failed to append version %d to stream %s: %w", version, streamID, err)
	}
	return nil
}

func (s *postgres) HasCommand(ctx context.Context, commandID uuid.UUID) (bool, error) {
	return hasCommand(ctx, s.pool, commandID)
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func hasCommand(ctx context.Context, db queryer, commandID uuid.UUID) (bool, error) {
	var handled bool
	if err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM events WHERE command_id = $1)", commandID).Scan(&handled); err != nil {
		return false, fmt.Errorf("failed to look for the events of command %s: %w", commandID, err)
	}
	return handled, nil
}

func (s *postgres) EventsForStream(ctx context.Context, streamID string) ([]domain.Event, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+row.Columns+" FROM events WHERE stream_id = $1 ORDER BY version", streamID)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream %s: %w", streamID, err)
	}

	stored, err := row.Scan(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream %s: %w", streamID, err)
	}

	events := make([]domain.Event, 0, len(stored))
	for _, e := range stored {
		evt, err := e.ToDomain(s.registry)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}
