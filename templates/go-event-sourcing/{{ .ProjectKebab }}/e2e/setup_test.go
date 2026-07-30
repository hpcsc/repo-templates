//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/app"
	"github.com/stretchr/testify/require"
)

const defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/{{ .ProjectSnake }}_test?sslmode=disable"

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return defaultDatabaseURL
}

// newApp opens the real store and builds the same graph cmd builds. Each test
// gets its own account ids so they can share one database without colliding.
func newApp(t *testing.T, at time.Time) *app.App {
	t.Helper()

	ctx := context.Background()
	application, err := app.New(ctx, app.Config{
		DatabaseURL: databaseURL(),
		Now:         func() time.Time { return at },
	})
	require.NoError(t, err, "is postgres running? try: task db:up")

	t.Cleanup(application.Close)
	return application
}

func accountID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("acc-%s-%d", t.Name(), time.Now().UnixNano())
}
