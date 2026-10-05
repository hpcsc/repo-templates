//go:build unit

package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/hpcsc/go-event-sourcing-cli-project/internal/event"
	"github.com/hpcsc/go-event-sourcing-cli-project/internal/store"
	"github.com/hpcsc/go-event-sourcing-cli-project/internal/stream"
	"github.com/stretchr/testify/require"
)

func openSQLite(t *testing.T, path string) *store.SQLite {
	s, err := store.OpenSQLite(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func rawDB(t *testing.T, path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSQLite(t *testing.T) {
	ctx := context.Background()

	t.Run("append", func(t *testing.T) {
		t.Run("keeps the events after the store closes", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.db")
			first, err := store.OpenSQLite(ctx, path)
			require.NoError(t, err)
			require.NoError(t, first.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a", Owner: "Ada"}))
			require.NoError(t, first.Close())

			records, seq, err := openSQLite(t, path).Load(ctx, "a")

			require.NoError(t, err)
			require.Equal(t, 1, seq)
			require.Equal(t, event.AccountOpened{AccountID: "a", Owner: "Ada"}, records[0].Event)
		})

		t.Run("returns a conflict when the stream is past the expected seq", func(t *testing.T) {
			s := openSQLite(t, filepath.Join(t.TempDir(), "events.db"))
			require.NoError(t, s.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"}))

			err := s.Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"})

			require.ErrorIs(t, err, stream.ErrVersionConflict)
		})
	})

	t.Run("change a saved event", func(t *testing.T) {
		t.Run("the table refuses an update and a delete", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.db")
			require.NoError(t, openSQLite(t, path).Append(ctx, "a", 0, event.AccountOpened{AccountID: "a"}))
			db := rawDB(t, path)

			_, updateErr := db.Exec(`UPDATE events SET data = '{}'`)
			_, deleteErr := db.Exec(`DELETE FROM events`)

			require.ErrorContains(t, updateErr, "events are append-only")
			require.ErrorContains(t, deleteErr, "events are append-only")
		})
	})

	t.Run("open", func(t *testing.T) {
		t.Run("refuses a store whose applied schema file has other contents now", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.db")
			require.NoError(t, openSQLite(t, path).Close())
			_, err := rawDB(t, path).Exec(`UPDATE schema_migrations SET content_hash = 'other' WHERE file = '0001_init.sql'`)
			require.NoError(t, err)

			_, err = store.OpenSQLite(ctx, path)

			require.ErrorIs(t, err, store.ErrSchemaChanged)
		})

		t.Run("refuses a store that has a schema file this build does not have", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.db")
			require.NoError(t, openSQLite(t, path).Close())
			_, err := rawDB(t, path).Exec(`INSERT INTO schema_migrations (file, content_hash) VALUES ('0002_from_a_newer_build.sql', 'x')`)
			require.NoError(t, err)

			_, err = store.OpenSQLite(ctx, path)

			require.ErrorIs(t, err, store.ErrNewerStore)
		})
	})
}
