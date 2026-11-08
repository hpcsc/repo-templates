package reaction

import (
	"context"
	"errors"

	"{{ .ModulePath }}/internal/domain"
	"{{ .ModulePath }}/internal/domain/event"
)

type conflictRetry struct {
	inner    Interface
	attempts int
}

func NewConflictRetry(inner Interface, attempts int) Interface {
	if attempts < 1 {
		attempts = 1
	}
	return &conflictRetry{inner: inner, attempts: attempts}
}

var _ Interface = (*conflictRetry)(nil)

func (r *conflictRetry) Name() string         { return r.inner.Name() }
func (r *conflictRetry) EventTypes() []string { return r.inner.EventTypes() }

func (r *conflictRetry) Handle(ctx context.Context, evt *domain.Event) error {
	for attempt := 0; attempt < r.attempts; attempt++ {
		err := r.inner.Handle(ctx, evt)
		if !errors.Is(err, event.ErrConcurrencyConflict) {
			return err
		}
	}
	return event.ErrConcurrencyConflict
}