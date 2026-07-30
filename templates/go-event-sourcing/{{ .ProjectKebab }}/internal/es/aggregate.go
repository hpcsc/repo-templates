package es

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// Retries are spread rather than immediate. Writers that collide would
// otherwise reload and collide again in lockstep, so under real contention the
// last one in the queue exhausts its budget losing the same race repeatedly.
const (
	retryBaseDelay = 2 * time.Millisecond
	retryMaxDelay  = 50 * time.Millisecond
)

// DefaultRetryAttempts is the budget a use case should pass unless it has a
// reason not to. It has to exceed the number of writers realistically racing
// for one stream, because each round of the race lets only one of them past.
const DefaultRetryAttempts = 10

// Root constrains a pointer to an aggregate state that can rebuild itself from
// its own history.
type Root[T any] interface {
	*T
	Apply(event Event)
}

type Repository[T any, A Root[T]] struct {
	store    Store
	attempts int
}

// NewRepository returns a repository that retries a decision when a concurrent
// append moved the stream underneath it. The loser of that race is stale
// rather than wrong, so deciding again against fresh history is correct.
func NewRepository[T any, A Root[T]](store Store, attempts int) Repository[T, A] {
	if attempts < 1 {
		attempts = 1
	}
	return Repository[T, A]{store: store, attempts: attempts}
}

// Update loads the stream, rebuilds the aggregate, asks it to decide, and
// appends whatever it returns. A decision that returns no events writes
// nothing, which is how an aggregate reports "already done".
func (r Repository[T, A]) Update(ctx context.Context, id string, decide func(A) ([]Event, error)) error {
	var err error
	for attempt := range r.attempts {
		err = r.update(ctx, id, decide)
		if !errors.Is(err, ErrVersionConflict) {
			return err
		}
		if attempt < r.attempts-1 {
			if waited := pause(ctx, attempt); waited != nil {
				return waited
			}
		}
	}
	return err
}

func pause(ctx context.Context, attempt int) error {
	delay := retryBaseDelay << min(attempt, 16)
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay + rand.N(delay)):
		return nil
	}
}

func (r Repository[T, A]) update(ctx context.Context, id string, decide func(A) ([]Event, error)) error {
	history, version, err := r.store.Load(ctx, id)
	if err != nil {
		return err
	}

	root := A(new(T))
	for _, e := range history {
		root.Apply(e)
	}

	events, err := decide(root)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	return r.store.Append(ctx, id, version, events)
}
