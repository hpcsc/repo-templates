//go:build unit

package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/stretchr/testify/require"
)

func TestBus(t *testing.T) {
	t.Run("dispatch", func(t *testing.T) {
		t.Run("gives the command to the handler of its payload type", func(t *testing.T) {
			b := command.NewBus()
			handler := fake.NewHandler[fake.Payload]()
			b.Register[fake.Payload](handler)
			cmd := domain.NewCommand(fake.Payload{Value: "value"})

			err := b.Dispatch(context.Background(), cmd)

			require.NoError(t, err)
			require.Equal(t, &cmd, handler.TriggeredWithCommand())
		})

		t.Run("returns an error when no handler has the payload type", func(t *testing.T) {
			b := command.NewBus()

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.ErrorContains(t, err, "no command handler registered for command fake.Payload")
		})

		t.Run("returns the error of the handler", func(t *testing.T) {
			b := command.NewBus()
			b.Register[fake.Payload](fake.NewHandler[fake.Payload]().WithError(errors.New("some error")))

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.ErrorContains(t, err, "some error")
		})

		t.Run("calls the handler again when another change saved first", func(t *testing.T) {
			b := command.NewBus()
			conflict := fmt.Errorf("%w: stream moved", event.ErrConcurrencyConflict)
			handler := fake.NewHandler[fake.Payload]().WithErrors(conflict, nil)
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.NoError(t, err)
			require.Equal(t, 2, handler.Calls())
		})

		t.Run("returns the conflict after the last attempt", func(t *testing.T) {
			b := command.NewBus()
			conflict := fmt.Errorf("%w: stream moved", event.ErrConcurrencyConflict)
			handler := fake.NewHandler[fake.Payload]().WithError(conflict)
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.ErrorIs(t, err, event.ErrConcurrencyConflict)
			require.Equal(t, 3, handler.Calls())
		})

		t.Run("does not call the handler when the context ended", func(t *testing.T) {
			b := command.NewBus()
			handler := fake.NewHandler[fake.Payload]()
			b.Register[fake.Payload](handler)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := b.Dispatch(ctx, domain.NewCommand(fake.Payload{}))

			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, 0, handler.Calls())
		})
	})
}
