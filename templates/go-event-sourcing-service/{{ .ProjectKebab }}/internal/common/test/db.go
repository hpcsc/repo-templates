//go:build integration

package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func DBConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@127.0.0.1:%s/%s?sslmode=disable",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST_PORT"),
		os.Getenv("DB_NAME"),
	)
}

func NewDBPool(t *testing.T) *pgxpool.Pool {
	require.NotEmpty(t, os.Getenv("DB_HOST_PORT"), "DB_HOST_PORT must be set, see .env.integration")

	pool, err := pgxpool.New(context.Background(), DBConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}
