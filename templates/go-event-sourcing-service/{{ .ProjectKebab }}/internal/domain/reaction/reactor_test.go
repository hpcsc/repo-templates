//go:build unit

package reaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/stretchr/testify/require"
)

func TestReactor(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("starts after the last event in the store and saves that position when no checkpoint exists", func(t *testing.T) {
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().
				WithEventChannel(closedChannel()).
				WithLastPosition(&domain.Position{TransactionID: 750, Sequence: 12})
			r := reaction.NewReactor(reaction.NewFake("test-reaction", nil, nil), checkpointStore, sub, test.DiscardLogger())

			_ = r.Start(context.Background())

			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, sub.CalledAfter)
			cp, err := checkpointStore.Get(context.Background(), "test-reaction")
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 12}, cp)
		})

		t.Run("starts at the beginning when told to, although the store has events", func(t *testing.T) {
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().
				WithEventChannel(closedChannel()).
				WithLastPosition(&domain.Position{TransactionID: 750, Sequence: 12})
			r := reaction.NewReactor(reaction.NewFake("test-reaction", nil, nil), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning())

			_ = r.Start(context.Background())

			require.Nil(t, sub.CalledAfter)
			cp, err := checkpointStore.Get(context.Background(), "test-reaction")
			require.NoError(t, err)
			require.Nil(t, cp)
		})

		t.Run("starts at the beginning when no checkpoint exists and the store is empty", func(t *testing.T) {
			sub := subscription.NewFake().WithEventChannel(closedChannel()).WithLastPosition(nil)
			r := reaction.NewReactor(reaction.NewFake("test-reaction", nil, nil), store.NewEmptyMemory(), sub, test.DiscardLogger())

			_ = r.Start(context.Background())

			require.Nil(t, sub.CalledAfter)
		})
	})

	t.Run("handle", func(t *testing.T) {
		t.Run("saves the checkpoint after each event, before it handles the next one", func(t *testing.T) {
			checkpointStore := store.NewMemory(map[string]domain.Position{"test-reaction": {TransactionID: 750, Sequence: 0}})
			var checkpointAtSecondEvent *domain.Position
			react := reaction.NewFake("test-reaction", nil, func(ctx context.Context, evt *domain.Event) error {
				if evt.Position.Sequence == 2 {
					checkpointAtSecondEvent, _ = checkpointStore.Get(ctx, "test-reaction")
				}
				return nil
			})
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1), eventAt(2)))

			err := reaction.NewReactor(react, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, event.ErrSubscriptionEnded)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 1}, checkpointAtSecondEvent)
		})

		t.Run("stops at the event that fails and keeps the checkpoint of the event before it", func(t *testing.T) {
			expectedErr := errors.New("side effect failed")
			checkpointStore := store.NewMemory(map[string]domain.Position{"test-reaction": {TransactionID: 750, Sequence: 0}})
			react := reaction.NewFake("test-reaction", nil, func(_ context.Context, evt *domain.Event) error {
				if evt.Position.Sequence == 2 {
					return expectedErr
				}
				return nil
			})
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1), eventAt(2), eventAt(3)))

			err := reaction.NewReactor(react, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
			cp, err := checkpointStore.Get(context.Background(), "test-reaction")
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 1}, cp)
		})
	})
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
