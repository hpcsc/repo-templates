//go:build unit

package projection_test

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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/stretchr/testify/require"
)

func TestProjector(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("starts at the beginning of the store when no checkpoint exists", func(t *testing.T) {
			sub := subscription.NewFake().WithEventChannel(closedChannel())
			p := projection.NewProjector(projection.NewMemory("test-projection"), projection.NewFakeDB(), store.NewEmptyMemory(), sub, test.DiscardLogger())

			_ = p.Start(context.Background())

			require.Nil(t, sub.CalledAfter)
		})
	})

	t.Run("handle", func(t *testing.T) {
		t.Run("commits each batch and its checkpoint in one transaction", func(t *testing.T) {
			proj := projection.NewMemory("test-projection")
			db := projection.NewFakeDB()
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1), eventAt(2), eventAt(3)))
			p := projection.NewProjector(proj, db, checkpointStore, sub, test.DiscardLogger(), projection.WithBatchSize(2))

			err := p.Start(context.Background())

			require.ErrorIs(t, err, event.ErrSubscriptionEnded)
			require.Len(t, proj.HandledEvents(), 3)
			require.Equal(t, 2, db.Committed)
			cp, err := checkpointStore.Get(context.Background(), "test-projection")
			require.NoError(t, err)
			require.Equal(t, &domain.Position{TransactionID: 750, Sequence: 3}, cp)
		})

		t.Run("rolls back the batch when the projection fails", func(t *testing.T) {
			expectedErr := errors.New("handle failed")
			db := projection.NewFakeDB()
			checkpointStore := store.NewEmptyMemory()
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1)))
			p := projection.NewProjector(projection.NewMemory("test-projection").WithHandleError(expectedErr), db, checkpointStore, sub, test.DiscardLogger())

			err := p.Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, 0, db.Committed)
			require.Equal(t, 1, db.RolledBack)
			cp, err := checkpointStore.Get(context.Background(), "test-projection")
			require.NoError(t, err)
			require.Nil(t, cp)
		})

		t.Run("rolls back the batch when the checkpoint cannot be saved", func(t *testing.T) {
			expectedErr := errors.New("checkpoint save failed")
			db := projection.NewFakeDB()
			sub := subscription.NewFake().WithEventChannel(closedChannel(eventAt(1)))
			p := projection.NewProjector(projection.NewMemory("test-projection"), db, store.NewBroken().WithSetError(expectedErr), sub, test.DiscardLogger())

			err := p.Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, 0, db.Committed)
			require.Equal(t, 1, db.RolledBack)
		})
	})
}

func eventAt(sequence uint64) domain.Event {
	return domain.Event{Payload: &fake.SomethingHappened{}, Position: domain.Position{TransactionID: 750, Sequence: sequence}}
}
