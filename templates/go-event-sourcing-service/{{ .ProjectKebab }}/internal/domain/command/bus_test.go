//go:build unit

package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	domainFake "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
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

	t.Run("repeated command", func(t *testing.T) {
		t.Run("does not call the handler for a command that the store has", func(t *testing.T) {
			history := stream.NewMemory()
			cmd := domain.NewCommand(fake.Payload{})
			saved := domainFake.NewSomethingHappenedEvent("1", "first")
			saved.CommandID = cmd.ID
			require.NoError(t, history.Save(context.Background(), "stream-1", []domain.Event{saved}, 0))
			b := command.NewBus(command.WithHistory(history))
			handler := fake.NewHandler[fake.Payload]()
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), cmd)

			require.NoError(t, err)
			require.Equal(t, 0, handler.Calls())
		})

		t.Run("calls the handler for a command that the store does not have", func(t *testing.T) {
			b := command.NewBus(command.WithHistory(stream.NewMemory()))
			handler := fake.NewHandler[fake.Payload]()
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.NoError(t, err)
			require.Equal(t, 1, handler.Calls())
		})

		t.Run("returns no error when another process saves the same command first", func(t *testing.T) {
			b := command.NewBus(command.WithHistory(stream.NewMemory()))
			duplicate := fmt.Errorf("%w: command saved", event.ErrDuplicateCommand)
			handler := fake.NewHandler[fake.Payload]().WithError(duplicate)
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.NoError(t, err)
			require.Equal(t, 1, handler.Calls())
		})

		t.Run("returns the error when the store cannot be read", func(t *testing.T) {
			expectedErr := errors.New("read failed")
			b := command.NewBus(command.WithHistory(brokenHistory{err: expectedErr}))
			handler := fake.NewHandler[fake.Payload]()
			b.Register[fake.Payload](handler)

			err := b.Dispatch(context.Background(), domain.NewCommand(fake.Payload{}))

			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, 0, handler.Calls())
		})
	})
}

type brokenHistory struct {
	err error
}

func (h brokenHistory) HasCommand(context.Context, uuid.UUID) (bool, error) {
	return false, h.err
}
