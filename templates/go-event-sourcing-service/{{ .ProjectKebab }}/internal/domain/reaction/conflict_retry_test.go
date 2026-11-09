//go:build unit

package reaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/stretchr/testify/require"
)

func TestConflictRetry(t *testing.T) {
	t.Run("Handle", func(t *testing.T) {
		t.Run("succeeds on the first attempt", func(t *testing.T) {
			calls := 0
			inner := reaction.NewFake("test", nil, func(_ context.Context, _ *domain.Event) error {
				calls++
				return nil
			})

			err := reaction.NewConflictRetry(inner, 3).Handle(context.Background(), &domain.Event{})

			require.NoError(t, err)
			require.Equal(t, 1, calls)
		})

		t.Run("retries on concurrency conflict and succeeds", func(t *testing.T) {
			calls := 0
			inner := reaction.NewFake("test", nil, func(_ context.Context, _ *domain.Event) error {
				calls++
				if calls < 3 {
					return event.ErrConcurrencyConflict
				}
				return nil
			})

			err := reaction.NewConflictRetry(inner, 3).Handle(context.Background(), &domain.Event{})

			require.NoError(t, err)
			require.Equal(t, 3, calls)
		})

		t.Run("returns the concurrency conflict after exhausting retries", func(t *testing.T) {
			inner := reaction.NewFake("test", nil, func(_ context.Context, _ *domain.Event) error {
				return event.ErrConcurrencyConflict
			})

			err := reaction.NewConflictRetry(inner, 2).Handle(context.Background(), &domain.Event{})

			require.ErrorIs(t, err, event.ErrConcurrencyConflict)
		})

		t.Run("does not retry on a non-concurrency error", func(t *testing.T) {
			boom := errors.New("boom")
			calls := 0
			inner := reaction.NewFake("test", nil, func(_ context.Context, _ *domain.Event) error {
				calls++
				return boom
			})

			err := reaction.NewConflictRetry(inner, 3).Handle(context.Background(), &domain.Event{})

			require.ErrorIs(t, err, boom)
			require.Equal(t, 1, calls)
		})
	})

	t.Run("Name", func(t *testing.T) {
		inner := reaction.NewFake("thumbnail", nil, nil)
		require.Equal(t, "thumbnail", reaction.NewConflictRetry(inner, 3).Name())
	})

	t.Run("EventTypes", func(t *testing.T) {
		types := []string{"image.captured", "link.captured"}
		inner := reaction.NewFake("test", types, nil)
		require.Equal(t, types, reaction.NewConflictRetry(inner, 3).EventTypes())
	})
}