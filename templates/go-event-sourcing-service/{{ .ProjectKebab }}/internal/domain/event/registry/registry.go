package registry

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

var (
	ErrUnknownEventType   = errors.New("unknown event type")
	ErrNewerSchemaVersion = errors.New("schema version is newer than this build reads")
)

type Upcast func(schemaVersion int, rawPayload []byte) ([]byte, error)

type RegisterOption func(*registration)

func WithSchemaVersion(schemaVersion int, upcast Upcast) RegisterOption {
	return func(r *registration) {
		r.schemaVersion = schemaVersion
		r.upcast = upcast
	}
}

type registration struct {
	newPayload    func() domain.EventPayload
	schemaVersion int
	upcast        Upcast
}

type OfEvents struct {
	registrationByType map[string]registration
}

func New() *OfEvents {
	return &OfEvents{
		registrationByType: make(map[string]registration),
	}
}

func (r *OfEvents) Register[P any, PP interface {
	*P
	domain.EventPayload
}](opts ...RegisterOption) {
	reg := registration{
		newPayload: func() domain.EventPayload {
			return PP(new(P))
		},
		schemaVersion: 1,
	}

	for _, opt := range opts {
		opt(&reg)
	}

	r.registrationByType[PP(new(P)).EventType()] = reg
}

func (r *OfEvents) SchemaVersionOf(eventType string) (int, error) {
	reg, err := r.registrationOf(eventType)
	if err != nil {
		return 0, err
	}
	return reg.schemaVersion, nil
}

func (r *OfEvents) ParseRawData(eventType string, schemaVersion int, rawPayload []byte, rawMetadata []byte) (domain.EventPayload, *domain.EventMetadata, error) {
	reg, err := r.registrationOf(eventType)
	if err != nil {
		return nil, nil, err
	}

	if schemaVersion > reg.schemaVersion {
		return nil, nil, fmt.Errorf("%w: event %s has schema version %d, and this build reads up to %d", ErrNewerSchemaVersion, eventType, schemaVersion, reg.schemaVersion)
	}

	if schemaVersion < reg.schemaVersion && reg.upcast != nil {
		if rawPayload, err = reg.upcast(schemaVersion, rawPayload); err != nil {
			return nil, nil, fmt.Errorf("failed to upcast event %s from schema version %d: %w", eventType, schemaVersion, err)
		}
	}

	payload := reg.newPayload()
	if err := json.Unmarshal(rawPayload, payload); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal payload for event %s: %w", eventType, err)
	}

	var metadata domain.EventMetadata
	if rawMetadata != nil {
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal metadata for event %s: %w", eventType, err)
		}

		return payload, &metadata, nil
	}

	return payload, nil, nil
}

func (r *OfEvents) registrationOf(eventType string) (registration, error) {
	reg, exist := r.registrationByType[eventType]
	if !exist {
		return registration{}, fmt.Errorf("no payload registered for event type %s: %w", eventType, ErrUnknownEventType)
	}
	return reg, nil
}
