//go:build unit

package scheduler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/scheduler"
	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	t.Run("tick", func(t *testing.T) {
		t.Run("ticks at each interval with a later time each time", func(t *testing.T) {
			scheduled := &fakeScheduled{}
			ctx, cancel := context.WithCancel(context.Background())
			ended := start(ctx, scheduler.New(scheduled, test.DiscardLogger(), scheduler.WithInterval(time.Millisecond)))

			require.Eventually(t, func() bool { return len(scheduled.tickTimes()) >= 3 }, 5*time.Second, time.Millisecond)
			cancel()

			require.NoError(t, <-ended)
			times := scheduled.tickTimes()
			require.True(t, times[0].Before(times[1]) && times[1].Before(times[2]), "the ticks are not in time order: %v", times)
		})

		t.Run("stops with the error of a tick", func(t *testing.T) {
			expectedErr := errors.New("read failed")
			scheduled := &fakeScheduled{err: expectedErr}

			err := scheduler.New(scheduled, test.DiscardLogger(), scheduler.WithInterval(time.Millisecond)).Start(context.Background())

			require.ErrorIs(t, err, expectedErr)
		})
	})

	t.Run("end", func(t *testing.T) {
		t.Run("returns no error and does not tick when the context ends first", func(t *testing.T) {
			scheduled := &fakeScheduled{}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := scheduler.New(scheduled, test.DiscardLogger(), scheduler.WithInterval(time.Hour)).Start(ctx)

			require.NoError(t, err)
			require.Empty(t, scheduled.tickTimes())
		})
	})
}

type fakeScheduled struct {
	err error

	mutex sync.Mutex
	times []time.Time
}

func (s *fakeScheduled) Name() string {
	return "test-scheduler"
}

func (s *fakeScheduled) Tick(_ context.Context, now time.Time) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.times = append(s.times, now)
	return s.err
}

func (s *fakeScheduled) tickTimes() []time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]time.Time(nil), s.times...)
}

func start(ctx context.Context, s *scheduler.Scheduler) <-chan error {
	ended := make(chan error, 1)
	go func() { ended <- s.Start(ctx) }()
	return ended
}
