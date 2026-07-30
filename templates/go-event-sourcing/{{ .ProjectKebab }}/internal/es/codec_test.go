//go:build unit

package es_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/stretchr/testify/require"
)

type unregistered struct{}

func (unregistered) EventName() string { return "es_test.unregistered" }

func TestCodec(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		t.Run("decodes back to the value that was encoded", func(t *testing.T) {
			name, schemaVersion, data, err := es.Encode(recorded{Value: "hello"})
			require.NoError(t, err)
			require.Equal(t, "es_test.recorded", name)
			require.Equal(t, 1, schemaVersion)

			decoded, err := es.Decode(name, schemaVersion, data)

			require.NoError(t, err)
			require.Equal(t, recorded{Value: "hello"}, decoded)
		})

		t.Run("refuses to encode a type nobody registered", func(t *testing.T) {
			_, _, _, err := es.Encode(unregistered{})

			require.ErrorContains(t, err, "unregistered")
		})

		t.Run("refuses to decode a name nobody registered", func(t *testing.T) {
			_, err := es.Decode("es_test.unregistered", 1, json.RawMessage(`{}`))

			require.ErrorContains(t, err, "unregistered")
		})
	})
}
