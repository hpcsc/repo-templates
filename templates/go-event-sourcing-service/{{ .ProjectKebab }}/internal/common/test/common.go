package test

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func NewID(t *testing.T) uuid.UUID {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	return id
}

func NewStringID(t *testing.T) string {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	return id.String()
}

func UnmarshalJson[T any](t *testing.T, j []byte) *T {
	var unmarshalled T
	require.NoError(t, json.Unmarshal(j, &unmarshalled))
	return &unmarshalled
}

func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
