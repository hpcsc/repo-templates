package fake

import (
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

var _ domain.Aggregate = (*Aggregate)(nil)

type Aggregate struct {
	domain.AggregateBase

	Value string
}

func (a *Aggregate) DoSomething(id string, values ...string) {
	for _, value := range values {
		a.record(NewSomethingHappenedEvent(id, value))
	}
}

func (a *Aggregate) Apply(e domain.Event) {
	if payload, ok := e.Payload.(*SomethingHappened); ok {
		a.Value = payload.Value
	}
}

func (a *Aggregate) record(e domain.Event) {
	a.Apply(e)
	a.Record(e)
}
