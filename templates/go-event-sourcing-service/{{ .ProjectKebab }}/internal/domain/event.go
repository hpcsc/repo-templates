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

func (p Position) Before(other Position) bool {
	if p.TransactionID != other.TransactionID {
		return p.TransactionID < other.TransactionID
	}
	return p.Sequence < other.Sequence
}

type EventPayload interface {
	EventType() string
}

type Event struct {
	ID        string
	StreamID  string
	Version   uint64
	Position  Position
	CommandID uuid.UUID
	Payload   EventPayload
	Metadata  *EventMetadata
}

func (e Event) Type() string {
	return e.Payload.EventType()
}

func (e Event) CausedBy(cause Cause) Event {
	e.CommandID = cause.CommandID
	e.Metadata = &EventMetadata{
		CorrelationID: cause.CorrelationID,
		CausationID:   cause.CausationID,
	}
	return e
}

func (e Event) CausedMetadata() EventMetadata {
	id, _ := uuid.Parse(e.ID)
	correlationID := id
	if e.Metadata != nil && e.Metadata.CorrelationID != uuid.Nil {
		correlationID = e.Metadata.CorrelationID
	}
	return EventMetadata{CorrelationID: correlationID, CausationID: id}
}

func NewEvent(payload EventPayload) Event {
	return Event{
		Payload: payload,
	}
}
