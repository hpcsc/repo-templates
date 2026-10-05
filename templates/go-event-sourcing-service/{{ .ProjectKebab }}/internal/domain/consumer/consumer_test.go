//go:build unit

package consumer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/consumer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("subscribes after the saved checkpoint and does not ask the handler", func(t *testing.T) {
			h := &fakeHandler{name: "test-consumer", start: &domain.Position{TransactionID: 1, Sequence: 1}}
			checkpointStore := store.NewMemory(map[string]domain.Position{"test-consumer": {TransactionID: 750, Sequence: 12}})
			sub := subscription.NewFake().WithEventChannel(closedChannel())

			_ = consumer.New(h, 10, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, sub.CalledAfter)
			require.False(t, h.startAsked)
		})

		t.Run("starts where the handler says when no checkpoint exists, and saves that position first", func(t *testing.T) {
			h := &fakeHandler{name: "test-consumer", start: &domain.Position{TransactionID: 750, Sequence: 12}}
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().WithEventChannel(closedChannel())

			_ = consumer.New(h, 10, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, sub.CalledAfter)
			saved, err := checkpointStore.Get(context.Background(), "test-consumer")
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, saved)
		})

		t.Run("starts at the beginning and saves nothing when the handler gives no position", func(t *testing.T) {
			h := &fakeHandler{name: "test-consumer"}
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().WithEventChannel(closedChannel())

			_ = consumer.New(h, 10, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.Nil(t, sub.CalledAfter)
			saved, err := checkpointStore.Get(context.Background(), "test-consumer")
			require.NoError(t, err)
			require.Nil(t, saved)
		})

		t.Run("returns the error when the checkpoint cannot be read", func(t *testing.T) {
			expectedErr := errors.New("checkpoint read failed")
			c := consumer.New(&fakeHandler{name: "test-consumer"}, 10, store.NewBroken().WithGetError(expectedErr), subscription.NewFake(), test.DiscardLogger())

			err := c.Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("returns the error when the handler cannot give a start position", func(t *testing.T) {
			expectedErr := errors.New("last position failed")
			h := &fakeHandler{name: "test-consumer", startErr: expectedErr}

			err := consumer.New(h, 10, store.NewEmptyMemory(), subscription.NewFake(), test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("returns the error when the subscription fails", func(t *testing.T) {
			expectedErr := errors.New("subscribe failed")
			sub := subscription.NewFake().WithSubscribeToAllError(expectedErr)

			err := consumer.New(&fakeHandler{name: "test-consumer"}, 10, store.NewEmptyMemory(), sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
		})
	})

	t.Run("batch", func(t *testing.T) {
		t.Run("gives the ready events to the handler in batches of the batch size", func(t *testing.T) {
			h := &fakeHandler{name: "test-consumer"}
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1), eventAt(2), eventAt(3), eventAt(4), eventAt(5)))

			err := consumer.New(h, 2, store.NewEmptyMemory(), sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, event.ErrSubscriptionEnded)
			require.Equal(t, []int{2, 2, 1}, h.batchSizes)
		})

		t.Run("returns the handler error and handles no more events", func(t *testing.T) {
			expectedErr := errors.New("handle failed")
			h := &fakeHandler{name: "test-consumer", handleErr: expectedErr}
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1), eventAt(2), eventAt(3)))

			err := consumer.New(h, 1, store.NewEmptyMemory(), sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, []int{1}, h.batchSizes)
		})
	})

	t.Run("end", func(t *testing.T) {
		t.Run("returns an error when the subscription ends before shutdown", func(t *testing.T) {
			sub := subscription.NewFake().WithEventChannel(closedChannel())

			err := consumer.New(&fakeHandler{name: "test-consumer"}, 10, store.NewEmptyMemory(), sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, event.ErrSubscriptionEnded)
		})

		t.Run("returns no error when the context ends", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := consumer.New(&fakeHandler{name: "test-consumer"}, 10, store.NewEmptyMemory(), subscription.NewFake(), test.DiscardLogger()).Start(ctx)

			require.NoError(t, err)
		})

		t.Run("returns no error when the subscription ends because of shutdown", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// select picks a ready case at random, so one run can miss the closed channel
			for range 20 {
				sub := subscription.NewFake().WithEventChannel(closedChannel())

				err := consumer.New(&fakeHandler{name: "test-consumer"}, 10, store.NewEmptyMemory(), sub, test.DiscardLogger()).Start(ctx)

				require.NoError(t, err)
			}
		})
	})
}

type fakeHandler struct {
	name       string
	start      *domain.Position
	startErr   error
	handleErr  error
	startAsked bool
	batchSizes []int
}

func (h *fakeHandler) Name() string {
	return h.name
}

func (h *fakeHandler) EventTypes() []string {
	return nil
}

func (h *fakeHandler) StartWithoutCheckpoint(context.Context, event.Subscription) (*domain.Position, error) {
	h.startAsked = true
	return h.start, h.startErr
}

func (h *fakeHandler) HandleBatch(_ context.Context, events []domain.Event) error {
	h.batchSizes = append(h.batchSizes, len(events))
	return h.handleErr
}

func eventAt(sequence uint64) domain.Event {
	return domain.Event{Payload: &fake.SomethingHappened{}, Position: domain.Position{TransactionID: 750, Sequence: sequence}}
}

func closedChannel(events ...domain.Event) <-chan domain.Event {
	ch := make(chan domain.Event, len(events))
	for _, evt := range events {
		ch <- evt
	}
	close(ch)
	return ch
}
