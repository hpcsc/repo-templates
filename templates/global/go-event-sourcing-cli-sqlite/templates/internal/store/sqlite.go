package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/hpcsc/go-event-sourcing-cli-project/internal/es"
	"github.com/hpcsc/go-event-sourcing-cli-project/internal/event"
	"github.com/hpcsc/go-event-sourcing-cli-project/internal/schema"
	"github.com/hpcsc/go-event-sourcing-cli-project/internal/stream"
	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

var (
	ErrSchemaChanged = errors.New("a schema file changed after the store applied it")
	ErrNewerStore    = errors.New("a newer build wrote this store")
)

var _ stream.Store = (*SQLite)(nil)

type SQLite struct {
	db *sql.DB
}

func OpenSQLite(ctx context.Context, path string) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create the directory of the store %q: %w", path, err)
	}

	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open the store %q: %w", path, err)
	}

	if err := applySchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open the store %q: %w", path, err)
	}

	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) Load(ctx context.Context, streamID string) ([]es.Record, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT seq, at, kind, v, data FROM events WHERE stream_id = ? ORDER BY seq`, streamID)
	if err != nil {
		return nil, 0, fmt.Errorf("load stream %q: %w", streamID, err)
	}
	defer rows.Close()

	var (
		records []es.Record
		seq     int
	)
	for rows.Next() {
		var (
			at, kind      string
			schemaVersion int
			data          []byte
		)
		if err := rows.Scan(&seq, &at, &kind, &schemaVersion, &data); err != nil {
			return nil, 0, fmt.Errorf("scan an event of stream %q: %w", streamID, err)
		}

		recordedAt, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, 0, fmt.Errorf("parse the time of an event of stream %q: %w", streamID, err)
		}

		decoded, err := event.Decode(kind, schemaVersion, data)
		if err != nil {
			return nil, 0, fmt.Errorf("decode an event of stream %q: %w", streamID, err)
		}

		records = append(records, es.Record{StreamID: streamID, Seq: seq, At: recordedAt, Event: decoded})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("load stream %q: %w", streamID, err)
	}

	return records, seq, nil
}

func (s *SQLite) Append(ctx context.Context, streamID string, expectedSeq int, events ...es.Event) (err error) {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin an append to stream %q: %w", streamID, err)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("roll back an append to stream %q: %w", streamID, rbErr))
		}
	}()

	for i, e := range events {
		kind, schemaVersion, data, err := event.Encode(e)
		if err != nil {
			return fmt.Errorf("append to stream %q: %w", streamID, err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO events (stream_id, seq, kind, v, data) VALUES (?, ?, ?, ?, ?)`,
			streamID, expectedSeq+1+i, kind, schemaVersion, string(data)); err != nil {
			var sqliteErr *sqlite.Error
			if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlitelib.SQLITE_CONSTRAINT_UNIQUE {
				return fmt.Errorf("%w: stream %q already has seq %d", stream.ErrVersionConflict, streamID, expectedSeq+1+i)
			}
			return fmt.Errorf("append to stream %q: %w", streamID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit an append to stream %q: %w", streamID, err)
	}
	return nil
}

func applySchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		file         TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		applied_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := fs.Glob(schema.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("list the schema files: %w", err)
	}
	sort.Strings(names)

	recorded, err := recordedSchema(ctx, db)
	if err != nil {
		return err
	}
	for file := range recorded {
		if !slices.Contains(names, file) {
			return fmt.Errorf("%w: it has schema file %q, which this build does not have", ErrNewerStore, file)
		}
	}

	for _, name := range names {
		ddl, err := schema.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read schema file %q: %w", name, err)
		}
		digest := sha256.Sum256(ddl)
		contentHash := hex.EncodeToString(digest[:])

		hash, applied := recorded[name]
		switch {
		case !applied:
			if err := applySchemaFile(ctx, db, name, ddl, contentHash); err != nil {
				return err
			}
		case hash != contentHash:
			return fmt.Errorf("%w: %q, add a new file instead of a change to an applied one", ErrSchemaChanged, name)
		}
	}
	return nil
}

func recordedSchema(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT file, content_hash FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	recorded := map[string]string{}
	for rows.Next() {
		var file, hash string
		if err := rows.Scan(&file, &hash); err != nil {
			return nil, fmt.Errorf("read schema_migrations: %w", err)
		}
		recorded[file] = hash
	}
	return recorded, rows.Err()
}

func applySchemaFile(ctx context.Context, db *sql.DB, name string, ddl []byte, contentHash string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema file %q: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(ddl)); err != nil {
		return fmt.Errorf("apply schema file %q: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (file, content_hash) VALUES (?, ?)`, name, contentHash); err != nil {
		return fmt.Errorf("record schema file %q: %w", name, err)
	}
	return tx.Commit()
}

// the driver applies these pragmas to each connection that the pool opens
func dsn(path string) string {
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(wal)")
	q.Add("_pragma", "synchronous(normal)")
	u.RawQuery = q.Encode()
	return u.String()
}
