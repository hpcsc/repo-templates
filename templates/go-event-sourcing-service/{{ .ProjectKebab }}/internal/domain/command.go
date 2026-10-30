package domain

import "github.com/google/uuid"

type Command[P any] struct {
	ID            uuid.UUID
	CorrelationID uuid.UUID
	CausationID   uuid.UUID
	Payload       P
}

func NewCommand[P any](payload P) Command[P] {
	return Command[P]{
		ID:      uuid.New(),
		Payload: payload,
	}
}

type Cause struct {
	CommandID     uuid.UUID
	CorrelationID uuid.UUID
	CausationID   uuid.UUID
}

func (c Command[P]) Cause() Cause {
	cause := Cause{CommandID: c.ID, CorrelationID: c.CorrelationID, CausationID: c.CausationID}
	if cause.CorrelationID == uuid.Nil {
		cause.CorrelationID = c.ID
	}
	if cause.CausationID == uuid.Nil {
		cause.CausationID = c.ID
	}
	return cause
}
