package domain

import "github.com/google/uuid"

type Command[P any] struct {
	ID      uuid.UUID
	Payload P
}

func NewCommand[P any](payload P) Command[P] {
	return Command[P]{
		ID:      uuid.New(),
		Payload: payload,
	}
}
