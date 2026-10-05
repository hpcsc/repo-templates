//go:build unit

package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootWithSQLite(t *testing.T) {
	t.Run("show", func(t *testing.T) {
		t.Run("shows the account that earlier commands opened and credited", func(t *testing.T) {
			db := filepath.Join(t.TempDir(), "events.db")
			require.Equal(t, 0, run(t, "--db", db, "open", "--id", "acc-1", "--owner", "Ada").status)
			require.Equal(t, 0, run(t, "--db", db, "credit", "--id", "acc-1", "--amount", "500", "--ref", "invoice-7").status)
			require.Equal(t, 0, run(t, "--db", db, "credit", "--id", "acc-1", "--amount", "500", "--ref", "invoice-7").status)

			r := run(t, "--db", db, "show", "--id", "acc-1")

			require.Equal(t, 0, r.status)
			require.Contains(t, r.stdout, `"balance":500`)
		})
	})
}
