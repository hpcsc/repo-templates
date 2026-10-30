//go:build unit

package holds_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/holds"
	"github.com/stretchr/testify/require"
)

func TestHandlers(t *testing.T) {
	t.Run("place", func(t *testing.T) {
		t.Run("saves the hold once when the command arrives twice", func(t *testing.T) {
			bus, s := newBus()
			id, paymentID := uuid.New(), uuid.New()
			given(t, s, id, opened(id), deposited(id, 500))
			place := domain.NewCommand(accountCommand.PlaceHold{AccountID: id, PaymentID: paymentID, Amount: 300})

			require.NoError(t, bus.Dispatch(context.Background(), place))
			require.NoError(t, bus.Dispatch(context.Background(), place))

			require.Equal(t, []any{&accountEvent.HoldPlaced{AccountID: id, PaymentID: paymentID, Amount: 300}}, payloadsOf(t, s, id)[2:])
		})
	})

	t.Run("release", func(t *testing.T) {
		t.Run("saves the release of the held amount", func(t *testing.T) {
			bus, s := newBus()
			id, paymentID := uuid.New(), uuid.New()
			given(t, s, id, opened(id), deposited(id, 500), held(id, paymentID, 300))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.ReleaseHold{AccountID: id, PaymentID: paymentID}))

			require.NoError(t, err)
			require.Equal(t, []any{&accountEvent.HoldReleased{AccountID: id, PaymentID: paymentID, Amount: 300}}, payloadsOf(t, s, id)[3:])
		})
	})

	t.Run("capture", func(t *testing.T) {
		t.Run("saves the capture of the held amount", func(t *testing.T) {
			bus, s := newBus()
			id, paymentID := uuid.New(), uuid.New()
			given(t, s, id, opened(id), deposited(id, 500), held(id, paymentID, 300))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.CaptureHold{AccountID: id, PaymentID: paymentID}))

			require.NoError(t, err)
			require.Equal(t, []any{&accountEvent.HoldCaptured{AccountID: id, PaymentID: paymentID, Amount: 300}}, payloadsOf(t, s, id)[3:])
		})

		t.Run("returns the error and saves nothing for a payment with no hold", func(t *testing.T) {
			bus, s := newBus()
			id := uuid.New()
			given(t, s, id, opened(id), deposited(id, 500))

			err := bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.CaptureHold{AccountID: id, PaymentID: uuid.New()}))

			require.ErrorIs(t, err, account.ErrNoHold)
			require.Empty(t, payloadsOf(t, s, id)[2:])
		})
	})
}

func newBus() (*command.Bus, event.Stream) {
	s := stream.NewMemory()
	bus := command.NewBus()
	holds.Register(bus, account.NewRepository(s))
	return bus, s
}

func given(t *testing.T, s event.Stream, id uuid.UUID, history ...domain.Event) {
	require.NoError(t, s.Save(context.Background(), account.StreamID(id), history, 0))
}

func opened(id uuid.UUID) domain.Event {
	return domain.NewEvent(&accountEvent.AccountOpened{AccountID: id, Owner: "alice"})
}

func deposited(id uuid.UUID, amount int64) domain.Event {
	return domain.NewEvent(&accountEvent.MoneyDeposited{AccountID: id, Amount: amount})
}

func held(id uuid.UUID, paymentID uuid.UUID, amount int64) domain.Event {
	return domain.NewEvent(&accountEvent.HoldPlaced{AccountID: id, PaymentID: paymentID, Amount: amount})
}

func payloadsOf(t *testing.T, s event.Stream, id uuid.UUID) []any {
	events, err := s.EventsForStream(context.Background(), account.StreamID(id))
	require.NoError(t, err)
	payloads := make([]any, 0, len(events))
	for _, e := range events {
		payloads = append(payloads, e.Payload)
	}
	return payloads
}
