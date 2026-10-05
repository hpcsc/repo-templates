package projection

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/jackc/pgx/v5"
)

var _ Interface = (*Memory)(nil)

func NewMemory(name string, eventTypes ...string) *Memory {
	return &Memory{
		name:       name,
		eventTypes: eventTypes,
	}
}

type Memory struct {
	name          string
	eventTypes    []string
	handledEvents []domain.Event
	handleError   error
}

func (m *Memory) WithHandleError(err error) *Memory {
	m.handleError = err
	return m
}

func (m *Memory) Handle(_ context.Context, _ pgx.Tx, event *domain.Event) error {
	if m.handleError != nil {
		return m.handleError
	}
	m.handledEvents = append(m.handledEvents, *event)
	return nil
}

func (m *Memory) Name() string {
	return m.name
}

func (m *Memory) EventTypes() []string {
	return m.eventTypes
}

func (m *Memory) HandledEvents() []domain.Event {
	return m.handledEvents
}
