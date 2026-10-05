package command

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

const maxAttempts = 3

func NewBus() *Bus {
	return &Bus{
		handlerByPayload: make(map[reflect.Type]any),
	}
}

type Bus struct {
	handlerByPayload map[reflect.Type]any
	mutex            sync.RWMutex
}

func (b *Bus) Register[P any](handler Handler[P]) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.handlerByPayload[reflect.TypeFor[P]()] = handler
}

func (b *Bus) Dispatch[P any](ctx context.Context, cmd domain.Command[P]) error {
	b.mutex.RLock()
	handler, ok := b.handlerByPayload[reflect.TypeFor[P]()].(Handler[P])
	b.mutex.RUnlock()
	if !ok {
		return fmt.Errorf("no command handler registered for command %T", cmd.Payload)
	}

	var err error
	for range maxAttempts {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = handler.Handle(ctx, cmd); !errors.Is(err, event.ErrConcurrencyConflict) {
			return err
		}
	}
	return err
}
