package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

// Postgres is the durable event store. Open applies any schema the database
// has not seen, so a fresh database and an up-to-date one are both ready to
// use after the same call.
type Postgres struct {
	pool    *pgxpool.Pool
	applied []string
}

func OpenPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	applied, err := applySchema(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Postgres{pool: pool, applied: applied}, nil
}

func (s *Postgres) Close() { s.pool.Close() }

// AppliedSchema names the schema files this open executed, in order. It is
// empty for a database that was already up to date.
func (s *Postgres) AppliedSchema() []string { return s.applied }

func (s *Postgres) Load(ctx context.Context, streamID string) ([]es.Event, int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT version, type, schema_version, data FROM events WHERE stream_id = $1 ORDER BY version`,
		streamID)
	if err != nil {
		return nil, 0, fmt.Errorf("loading stream %q: %w", streamID, err)
	}
	defer rows.Close()

	var (
		history []es.Event
		version int
	)
	for rows.Next() {
		var (
			rowVersion    int
			name          string
			schemaVersion int
			data          json.RawMessage
		)
		if err := rows.Scan(&rowVersion, &name, &schemaVersion, &data); err != nil {
			return nil, 0, fmt.Errorf("scanning event from stream %q: %w", streamID, err)
		}

		event, err := es.Decode(name, schemaVersion, data)
		if err != nil {
			return nil, 0, fmt.Errorf("stream %q: %w", streamID, err)
		}

		history = append(history, event)
		version = rowVersion
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("loading stream %q: %w", streamID, err)
	}

	return history, version, nil
}

// Append writes the events as one transaction at expectedVersion+1 onwards. A
// unique violation on (stream_id, version) means another writer got there
// first, which surfaces as es.ErrVersionConflict for the repository to retry.
func (s *Postgres) Append(ctx context.Context, streamID string, expectedVersion int, events []es.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning append to stream %q: %w", streamID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i, event := range events {
		name, schemaVersion, data, err := es.Encode(event)
		if err != nil {
			return fmt.Errorf("stream %q: %w", streamID, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO events (stream_id, version, type, schema_version, data) VALUES ($1, $2, $3, $4, $5)`,
			streamID, expectedVersion+1+i, name, schemaVersion, data)
		if err != nil {
			return appendError(streamID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return appendError(streamID, err)
	}

	return nil
}

func appendError(streamID string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return fmt.Errorf("appending to stream %q: %w", streamID, es.ErrVersionConflict)
	}
	return fmt.Errorf("appending to stream %q: %w", streamID, err)
}

var _ es.Store = (*Postgres)(nil)

type schemaFile struct {
	name        string
	ddl         []byte
	contentHash string
}

func applySchema(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		file         TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		applied_at   TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return nil, fmt.Errorf("creating schema_migrations: %w", err)
	}

	pending, err := pendingSchema(ctx, pool)
	if err != nil {
		return nil, err
	}

	applied := make([]string, 0, len(pending))
	for _, file := range pending {
		if err := applySchemaFile(ctx, pool, file); err != nil {
			return nil, err
		}
		applied = append(applied, file.name)
	}

	return applied, nil
}

func pendingSchema(ctx context.Context, pool *pgxpool.Pool) ([]schemaFile, error) {
	names, err := fs.Glob(schema.FS, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("listing embedded schema: %w", err)
	}
	sort.Strings(names)

	var pending []schemaFile
	for _, name := range names {
		ddl, err := schema.FS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading embedded schema %q: %w", name, err)
		}

		digest := sha256.Sum256(ddl)
		file := schemaFile{name: name, ddl: ddl, contentHash: hex.EncodeToString(digest[:])}

		var recorded string
		err = pool.QueryRow(ctx, `SELECT content_hash FROM schema_migrations WHERE file = $1`, name).Scan(&recorded)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			pending = append(pending, file)
		case err != nil:
			return nil, fmt.Errorf("checking schema %q: %w", name, err)
		case recorded != file.contentHash:
			return nil, fmt.Errorf(
				"schema %q was applied with different contents: add a new file instead of editing an applied one", name)
		}
	}

	return pending, nil
}

func applySchemaFile(ctx context.Context, pool *pgxpool.Pool, file schemaFile) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning schema %q: %w", file.name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(file.ddl)); err != nil {
		return fmt.Errorf("applying schema %q: %w", file.name, err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO schema_migrations (file, content_hash) VALUES ($1, $2)`,
		file.name, file.contentHash)
	if err != nil {
		return fmt.Errorf("recording schema %q: %w", file.name, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing schema %q: %w", file.name, err)
	}

	return nil
}
