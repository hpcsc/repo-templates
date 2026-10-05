package domain

import "github.com/google/uuid"

type EventMetadata struct {
	CorrelationID uuid.UUID
	CausationID   uuid.UUID
}

type Position struct {
	TransactionID uint64
	Sequence      uint64
}

type EventPayload interface {
	EventType() string
}

type Event struct {
	ID       string
	StreamID string
	Version  uint64
	Position Position
	Payload  EventPayload
	Metadata *EventMetadata
}

func (e Event) Type() string {
	return e.Payload.EventType()
}

func (e Event) CorrelateWithCommandID(commandID uuid.UUID) Event {
	e.Metadata = &EventMetadata{
		CorrelationID: commandID,
		CausationID:   commandID,
	}
	return e
}

func NewEvent(payload EventPayload) Event {
	return Event{
		Payload: payload,
	}
}
