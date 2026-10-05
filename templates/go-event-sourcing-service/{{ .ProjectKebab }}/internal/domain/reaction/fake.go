package reaction

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

var _ Interface = (*Fake)(nil)

func NewFake(name string, eventTypes []string, handler func(ctx context.Context, event *domain.Event) error) *Fake {
	return &Fake{
		name:        name,
		eventTypes:  eventTypes,
		handlerFunc: handler,
	}
}

type Fake struct {
	name        string
	eventTypes  []string
	handlerFunc func(ctx context.Context, event *domain.Event) error
}

func (f *Fake) Handle(ctx context.Context, event *domain.Event) error {
	if f.handlerFunc != nil {
		return f.handlerFunc(ctx, event)
	}
	return nil
}

func (f *Fake) EventTypes() []string {
	return f.eventTypes
}

func (f *Fake) Name() string {
	return f.name
}
