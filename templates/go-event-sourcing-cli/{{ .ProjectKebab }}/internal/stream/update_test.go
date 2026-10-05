//go:build unit

package stream_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
	"github.com/stretchr/testify/require"
)

type counter struct {
	records int
}

func (c *counter) Apply(es.Record) {
	c.records++
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("update", func(t *testing.T) {
		t.Run("decides on the folded stream and appends at the loaded seq", func(t *testing.T) {
			s := store.NewMemory()
			require.NoError(t, s.Append(ctx, "s", 0, event.AccountOpened{AccountID: "acc-1"}))
			var seen int

			err := stream.Update(ctx, s, "s", func(c *counter) ([]es.Event, error) {
				seen = c.records
				return []es.Event{event.AccountCredited{AccountID: "acc-1", Amount: 5}}, nil
			})

			require.NoError(t, err)
			require.Equal(t, 1, seen)
			records, seq, err := s.Load(ctx, "s")
			require.NoError(t, err)
			require.Equal(t, 2, seq)
			require.Equal(t, event.AccountCredited{AccountID: "acc-1", Amount: 5}, records[1].Event)
		})

		t.Run("appends nothing when decide returns nothing", func(t *testing.T) {
			s := store.NewMemory()

			err := stream.Update(ctx, s, "s", func(*counter) ([]es.Event, error) {
				return nil, nil
			})

			require.NoError(t, err)
			_, seq, err := s.Load(ctx, "s")
			require.NoError(t, err)
			require.Equal(t, 0, seq)
		})

		t.Run("returns the decide error once, and appends nothing", func(t *testing.T) {
			s := store.NewMemory()
			refused := errors.New("refused")
			calls := 0

			err := stream.Update(ctx, s, "s", func(*counter) ([]es.Event, error) {
				calls++
				return []es.Event{event.AccountOpened{AccountID: "acc-1"}}, refused
			})

			require.ErrorIs(t, err, refused)
			require.Equal(t, 1, calls)
			_, seq, err := s.Load(ctx, "s")
			require.NoError(t, err)
			require.Equal(t, 0, seq)
		})

		t.Run("reads the stream again and appends at the new seq when the stream moved once", func(t *testing.T) {
			s := store.NewMemory()
			var seen []int

			err := stream.Update(ctx, s, "s", func(c *counter) ([]es.Event, error) {
				seen = append(seen, c.records)
				if len(seen) == 1 {
					require.NoError(t, s.Append(ctx, "s", 0, event.AccountOpened{AccountID: "acc-1"}))
				}
				return []es.Event{event.AccountCredited{AccountID: "acc-1", Amount: 5}}, nil
			})

			require.NoError(t, err)
			require.Equal(t, []int{0, 1}, seen)
			_, seq, err := s.Load(ctx, "s")
			require.NoError(t, err)
			require.Equal(t, 2, seq)
		})

		t.Run("returns a conflict when the stream moves again before the second append", func(t *testing.T) {
			s := store.NewMemory()
			calls := 0

			err := stream.Update(ctx, s, "s", func(c *counter) ([]es.Event, error) {
				calls++
				require.NoError(t, s.Append(ctx, "s", c.records, event.AccountCredited{AccountID: "acc-1", Amount: 1}))
				return []es.Event{event.AccountCredited{AccountID: "acc-1", Amount: 5}}, nil
			})

			require.ErrorIs(t, err, stream.ErrVersionConflict)
			require.Equal(t, 2, calls)
		})
	})
}
