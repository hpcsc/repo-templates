package stream

import (
	"context"
	"errors"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

type State[S any] interface {
	*S
	Projection
}

func Update[S any, P State[S]](ctx context.Context, st Store, streamID string, decide func(state P) ([]es.Event, error)) error {
	err := updateOnce(ctx, st, streamID, decide)
	if errors.Is(err, ErrVersionConflict) {
		return updateOnce(ctx, st, streamID, decide)
	}
	return err
}

func updateOnce[S any, P State[S]](ctx context.Context, st Store, streamID string, decide func(state P) ([]es.Event, error)) error {
	history, seq, err := st.Load(ctx, streamID)
	if err != nil {
		return err
	}

	state := P(new(S))
	Fold(history, state)

	events, err := decide(state)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	return st.Append(ctx, streamID, seq, events...)
}
