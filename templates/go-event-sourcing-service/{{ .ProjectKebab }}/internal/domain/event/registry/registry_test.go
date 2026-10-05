//go:build unit

package registry_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	somethingHappened := fake.SomethingHappened{}.EventType()

	t.Run("schema version", func(t *testing.T) {
		t.Run("is 1 for a type that is registered with no schema version", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened]()

			schemaVersion, err := r.SchemaVersionOf(somethingHappened)

			require.NoError(t, err)
			require.Equal(t, 1, schemaVersion)
		})

		t.Run("is the schema version that the registration gives", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(3, nil))

			schemaVersion, err := r.SchemaVersionOf(somethingHappened)

			require.NoError(t, err)
			require.Equal(t, 3, schemaVersion)
		})

		t.Run("returns an unknown type error for a type that is not registered", func(t *testing.T) {
			_, err := registry.New().SchemaVersionOf(somethingHappened)

			require.ErrorIs(t, err, registry.ErrUnknownEventType)
		})
	})

	t.Run("parse raw data", func(t *testing.T) {
		t.Run("returns an unknown type error for a type that is not registered", func(t *testing.T) {
			r := registry.New()

			_, _, err := r.ParseRawData("{{ .ProjectKebab }}.not-registered", 1, []byte(`{}`), nil)

			require.ErrorContains(t, err, "no payload registered for event type {{ .ProjectKebab }}.not-registered")
			require.ErrorIs(t, err, registry.ErrUnknownEventType)
		})

		t.Run("returns a parse error, not an unknown type error, for data that does not fit the payload", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened]()

			_, _, err := r.ParseRawData(somethingHappened, 1, []byte("invalid-data"), nil)

			require.ErrorContains(t, err, "failed to unmarshal payload for event")
			require.False(t, errors.Is(err, registry.ErrUnknownEventType))
		})

		t.Run("returns a new payload of the struct that names the event type", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened]()

			payload, metadata, err := r.ParseRawData(somethingHappened, 1, []byte(`{"id": "1", "value": "value-1"}`), nil)

			require.NoError(t, err)
			require.Equal(t, &fake.SomethingHappened{ID: "1", Value: "value-1"}, payload)
			require.Nil(t, metadata)
		})

		t.Run("returns the metadata when the event has it", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened]()

			correlationID := test.NewID(t)
			causationID := test.NewID(t)
			payload, metadata, err := r.ParseRawData(
				somethingHappened,
				1,
				[]byte(`{"id": "1", "value": "value-1"}`),
				[]byte(fmt.Sprintf(`{"correlationID": "%s", "causationID": "%s"}`, correlationID, causationID)),
			)

			require.NoError(t, err)
			require.Equal(t, &fake.SomethingHappened{ID: "1", Value: "value-1"}, payload)
			require.Equal(t, &domain.EventMetadata{
				CorrelationID: correlationID,
				CausationID:   causationID,
			}, metadata)
		})

		t.Run("returns a newer schema version error for an event that a newer build saved", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(2, nil))

			_, _, err := r.ParseRawData(somethingHappened, 3, []byte(`{"id": "1", "value": "value-1"}`), nil)

			require.ErrorIs(t, err, registry.ErrNewerSchemaVersion)
			require.ErrorContains(t, err, "has schema version 3, and this build reads up to 2")
		})

		t.Run("upcasts the data of an older schema version before it parses the data", func(t *testing.T) {
			r := registry.New()
			var upcastFrom int
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(2, func(schemaVersion int, rawPayload []byte) ([]byte, error) {
				upcastFrom = schemaVersion
				var old struct{ ID, Text string }
				if err := json.Unmarshal(rawPayload, &old); err != nil {
					return nil, err
				}
				return json.Marshal(fake.SomethingHappened{ID: old.ID, Value: old.Text})
			}))

			payload, _, err := r.ParseRawData(somethingHappened, 1, []byte(`{"id": "1", "text": "value-1"}`), nil)

			require.NoError(t, err)
			require.Equal(t, 1, upcastFrom)
			require.Equal(t, &fake.SomethingHappened{ID: "1", Value: "value-1"}, payload)
		})

		t.Run("does not upcast the data of the current schema version", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(2, func(int, []byte) ([]byte, error) {
				return nil, errors.New("upcast of current data")
			}))

			payload, _, err := r.ParseRawData(somethingHappened, 2, []byte(`{"id": "1", "value": "value-1"}`), nil)

			require.NoError(t, err)
			require.Equal(t, &fake.SomethingHappened{ID: "1", Value: "value-1"}, payload)
		})

		t.Run("parses the data of an older schema version as it is when the registration has no upcast", func(t *testing.T) {
			r := registry.New()
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(2, nil))

			payload, _, err := r.ParseRawData(somethingHappened, 1, []byte(`{"id": "1", "value": "value-1"}`), nil)

			require.NoError(t, err)
			require.Equal(t, &fake.SomethingHappened{ID: "1", Value: "value-1"}, payload)
		})

		t.Run("returns the upcast error with the schema version that failed", func(t *testing.T) {
			r := registry.New()
			upcastErr := errors.New("field missing")
			r.Register[fake.SomethingHappened](registry.WithSchemaVersion(2, func(int, []byte) ([]byte, error) {
				return nil, upcastErr
			}))

			_, _, err := r.ParseRawData(somethingHappened, 1, []byte(`{}`), nil)

			require.ErrorIs(t, err, upcastErr)
			require.ErrorContains(t, err, "from schema version 1")
		})
	})
}
