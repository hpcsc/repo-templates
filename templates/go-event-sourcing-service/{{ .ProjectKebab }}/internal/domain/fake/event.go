package fake

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

type SomethingHappened struct {
	ID    string
	Value string
}

func (SomethingHappened) EventType() string {
	return event.Namespace + ".something-happened"
}

func NewSomethingHappenedEvent(id string, value string) domain.Event {
	return domain.NewEvent(&SomethingHappened{
		ID:    id,
		Value: value,
	})
}
