package command

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
)

const maxAttempts = 3

type History interface {
	HasCommand(ctx context.Context, commandID uuid.UUID) (bool, error)
}

type BusOption func(*Bus)

func WithHistory(history History) BusOption {
	return func(b *Bus) {
		b.history = history
	}
}

func NewBus(opts ...BusOption) *Bus {
	b := &Bus{
		handlerByPayload: make(map[reflect.Type]any),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

type Bus struct {
	handlerByPayload map[reflect.Type]any
	history          History
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
		if handled, err := b.handled(ctx, cmd.ID); err != nil || handled {
			return err
		}

		err = handler.Handle(ctx, cmd)
		if errors.Is(err, event.ErrDuplicateCommand) {
			return nil
		}
		if !errors.Is(err, event.ErrConcurrencyConflict) {
			return err
		}
	}
	return err
}

func (b *Bus) handled(ctx context.Context, commandID uuid.UUID) (bool, error) {
	if b.history == nil {
		return false, nil
	}

	handled, err := b.history.HasCommand(ctx, commandID)
	if err != nil {
		return false, fmt.Errorf("failed to check command %s: %w", commandID, err)
	}
	return handled, nil
}
