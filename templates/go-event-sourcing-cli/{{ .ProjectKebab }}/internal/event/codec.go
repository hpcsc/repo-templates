package event

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type decoder func(schemaVersion int, data json.RawMessage) (es.Event, error)

type registration struct {
	schemaVersion int
	decode        decoder
}

var registry = map[string]registration{}

func register[E es.Event]() {
	registerVersioned[E](1, func(_ int, data json.RawMessage) (es.Event, error) {
		var e E
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	})
}

func registerVersioned[E es.Event](currentVersion int, decode decoder) {
	var zero E
	if _, taken := registry[zero.Kind()]; taken {
		panic(fmt.Sprintf("event kind %q is registered twice", zero.Kind()))
	}
	registry[zero.Kind()] = registration{schemaVersion: currentVersion, decode: decode}
}

func Kinds() []string {
	kinds := make([]string, 0, len(registry))
	for kind := range registry {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

type NewerSchemaError struct {
	Kind      string
	Recorded  int
	Supported int
}

func (e *NewerSchemaError) Error() string {
	return fmt.Sprintf("decode %q: schema version %d was written by a newer build, which this one only reads up to version %d",
		e.Kind, e.Recorded, e.Supported)
}

func Encode(event es.Event) (kind string, schemaVersion int, data json.RawMessage, err error) {
	kind = event.Kind()
	reg, ok := registry[kind]
	if !ok {
		return "", 0, nil, fmt.Errorf("encode: unregistered event kind %q", kind)
	}

	data, err = json.Marshal(event)
	if err != nil {
		return "", 0, nil, fmt.Errorf("encode %q: %w", kind, err)
	}

	return kind, reg.schemaVersion, data, nil
}

func Decode(kind string, schemaVersion int, data json.RawMessage) (es.Event, error) {
	reg, ok := registry[kind]
	if !ok {
		return nil, fmt.Errorf("decode: unregistered event kind %q", kind)
	}

	if schemaVersion > reg.schemaVersion {
		return nil, &NewerSchemaError{Kind: kind, Recorded: schemaVersion, Supported: reg.schemaVersion}
	}

	return reg.decode(schemaVersion, data)
}
