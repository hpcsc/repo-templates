//go:build unit

package event_test

import (
	"errors"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	t.Run("encode", func(t *testing.T) {
		t.Run("reads back the event that it writes", func(t *testing.T) {
			opened := event.AccountOpened{AccountID: "acc-1", Owner: "Ada", OpenedAt: time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)}

			kind, schemaVersion, data, err := event.Encode(opened)
			require.NoError(t, err)
			decoded, err := event.Decode(kind, schemaVersion, data)

			require.NoError(t, err)
			require.Equal(t, "account.opened", kind)
			require.Equal(t, 1, schemaVersion)
			require.Equal(t, opened, decoded)
		})
	})

	t.Run("decode", func(t *testing.T) {
		t.Run("refuses a schema version that a newer build wrote", func(t *testing.T) {
			_, err := event.Decode("account.opened", 2, []byte(`{"account_id": "acc-1"}`))

			var newer *event.NewerSchemaError
			require.True(t, errors.As(err, &newer), "got %v", err)
			require.Equal(t, 2, newer.Recorded)
			require.Equal(t, 1, newer.Supported)
		})

		t.Run("refuses a kind that is not registered", func(t *testing.T) {
			_, err := event.Decode("account.renamed", 1, []byte(`{}`))

			require.ErrorContains(t, err, `unregistered event kind "account.renamed"`)
		})
	})

	t.Run("kinds", func(t *testing.T) {
		t.Run("lists every registered kind in order", func(t *testing.T) {
			require.Equal(t, []string{"account.credited", "account.opened"}, event.Kinds())
		})
	})
}
