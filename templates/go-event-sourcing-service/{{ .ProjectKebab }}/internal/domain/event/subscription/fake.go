package subscription

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

var _ event.Subscription = (*Fake)(nil)

func NewFake() *Fake {
	return &Fake{}
}

type Fake struct {
	eventChan           <-chan domain.Event
	subscribeToAllError error
	lastPosition        *domain.Position
	lastPositionError   error

	CalledAfter      *domain.Position
	CalledEventTypes []string
}

func (f *Fake) WithEventChannel(eventChan <-chan domain.Event) *Fake {
	f.eventChan = eventChan
	return f
}

func (f *Fake) WithSubscribeToAllError(err error) *Fake {
	f.subscribeToAllError = err
	return f
}

func (f *Fake) WithLastPosition(position *domain.Position) *Fake {
	f.lastPosition = position
	return f
}

func (f *Fake) WithLastPositionError(err error) *Fake {
	f.lastPositionError = err
	return f
}

func (f *Fake) SubscribeToAll(_ context.Context, after *domain.Position, eventTypes []string) (<-chan domain.Event, error) {
	f.CalledAfter = after
	f.CalledEventTypes = eventTypes

	if f.subscribeToAllError != nil {
		return nil, f.subscribeToAllError
	}

	if f.eventChan != nil {
		return f.eventChan, nil
	}

	return make(chan domain.Event), nil
}

func (f *Fake) LastPosition(context.Context) (*domain.Position, error) {
	return f.lastPosition, f.lastPositionError
}
