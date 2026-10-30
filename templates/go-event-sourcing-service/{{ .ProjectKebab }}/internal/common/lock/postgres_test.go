//go:build integration

package lock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/lock"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	t.Run("hold", func(t *testing.T) {
		t.Run("does not start while another process holds the lock of the name", func(t *testing.T) {
			locks := lock.NewPostgres(test.NewDBPool(t), lock.WithInterval(20*time.Millisecond))
			name := "test-" + test.NewStringID(t)
			release := holdLock(t, locks, name)
			defer release()
			waited, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			started := false

			err := locks.Hold(waited, name, func(context.Context) error {
				started = true
				return nil
			})

			require.NoError(t, err)
			require.False(t, started)
		})

		t.Run("starts when the other process releases the lock of the name", func(t *testing.T) {
			locks := lock.NewPostgres(test.NewDBPool(t), lock.WithInterval(20*time.Millisecond))
			name := "test-" + test.NewStringID(t)
			holdLock(t, locks, name)()
			started := false

			err := locks.Hold(context.Background(), name, func(context.Context) error {
				started = true
				return nil
			})

			require.NoError(t, err)
			require.True(t, started)
		})

		t.Run("starts while another process holds the lock of another name", func(t *testing.T) {
			locks := lock.NewPostgres(test.NewDBPool(t), lock.WithInterval(20*time.Millisecond))
			release := holdLock(t, locks, "test-"+test.NewStringID(t))
			defer release()
			waited, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			started := false

			err := locks.Hold(waited, "test-"+test.NewStringID(t), func(context.Context) error {
				started = true
				return nil
			})

			require.NoError(t, err)
			require.True(t, started)
		})

		t.Run("returns the error of start", func(t *testing.T) {
			locks := lock.NewPostgres(test.NewDBPool(t))
			expectedErr := errors.New("start failed")

			err := locks.Hold(context.Background(), "test-"+test.NewStringID(t), func(context.Context) error {
				return expectedErr
			})

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ends the context of start and returns ErrLost when the connection of the lock ends", func(t *testing.T) {
			pool := test.NewDBPool(t)
			locks := lock.NewPostgres(pool, lock.WithInterval(20*time.Millisecond))
			name := "test-" + test.NewStringID(t)
			started := make(chan struct{})
			ended := make(chan error, 1)
			go func() {
				ended <- locks.Hold(context.Background(), name, func(ctx context.Context) error {
					close(started)
					<-ctx.Done()
					return nil
				})
			}()
			<-started

			_, err := pool.Exec(context.Background(), "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name LIKE '%' || $1", name)
			require.NoError(t, err)

			select {
			case err := <-ended:
				require.ErrorIs(t, err, lock.ErrLost)
			case <-time.After(5 * time.Second):
				t.Fatal("start did not end after the lock ended")
			}
		})
	})
}

func holdLock(t *testing.T, locks *lock.Postgres, name string) (release func()) {
	started := make(chan struct{})
	stop := make(chan struct{})
	ended := make(chan error, 1)
	go func() {
		ended <- locks.Hold(context.Background(), name, func(context.Context) error {
			close(started)
			<-stop
			return nil
		})
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock was not taken")
	}

	return func() {
		close(stop)
		require.NoError(t, <-ended)
	}
}
