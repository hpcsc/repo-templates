package es

import (
	"encoding/json"
	"fmt"
	"sync"
)

// A stored event is (name, schema version, JSON). The name is what the type is
// called on disk, and it must outlive any Go type rename: change the struct
// name freely, but leave EventName alone or old streams stop decoding.
type decoder func(schemaVersion int, data json.RawMessage) (Event, error)

type registration struct {
	schemaVersion int
	decode        decoder
}

var (
	registryMu sync.RWMutex
	registry   = map[string]registration{}
)

// Register makes an event type decodable. Call it from an init in the package
// that declares the event, so importing that package is enough to teach the
// store about it.
func Register[E Event]() {
	RegisterVersioned[E](1, func(_ int, data json.RawMessage) (Event, error) {
		var e E
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	})
}

// RegisterVersioned is Register for a type whose stored shape has changed. The
// decoder receives the schema version each row was written with and is
// responsible for upgrading older ones to the current struct.
func RegisterVersioned[E Event](currentVersion int, decode decoder) {
	var zero E
	name := zero.EventName()

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, taken := registry[name]; taken {
		panic(fmt.Sprintf("es: event name %q registered twice", name))
	}
	registry[name] = registration{schemaVersion: currentVersion, decode: decode}
}

func Encode(event Event) (name string, schemaVersion int, data json.RawMessage, err error) {
	name = event.EventName()

	registryMu.RLock()
	reg, known := registry[name]
	registryMu.RUnlock()

	if !known {
		return "", 0, nil, fmt.Errorf("es: encoding unregistered event %q", name)
	}

	data, err = json.Marshal(event)
	if err != nil {
		return "", 0, nil, fmt.Errorf("es: encoding %q: %w", name, err)
	}

	return name, reg.schemaVersion, data, nil
}

func Decode(name string, schemaVersion int, data json.RawMessage) (Event, error) {
	registryMu.RLock()
	reg, known := registry[name]
	registryMu.RUnlock()

	if !known {
		return nil, fmt.Errorf("es: decoding unregistered event %q", name)
	}

	event, err := reg.decode(schemaVersion, data)
	if err != nil {
		return nil, fmt.Errorf("es: decoding %q at schema version %d: %w", name, schemaVersion, err)
	}

	return event, nil
}

// Registered lists the event names the process knows how to decode. A store
// can use it to fail loudly at startup rather than on the first bad read.
func Registered() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
