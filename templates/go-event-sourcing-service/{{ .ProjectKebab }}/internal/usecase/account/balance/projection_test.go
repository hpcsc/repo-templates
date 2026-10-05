//go:build integration

package balance_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/balance"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/welcomebonus"
	"github.com/stretchr/testify/require"
)

func TestBalance(t *testing.T) {
	t.Run("projection", func(t *testing.T) {
		t.Run("shows the balance after the deposits, the withdrawals and the welcome bonus, and the close", func(t *testing.T) {
			pool := test.NewDBPool(t)
			reg := registry.New()
			usecase.RegisterEvents(reg)
			bus := command.NewBus()
			usecase.RegisterHandlers(bus, stream.NewPostgres(pool, reg))
			checkpointStore := store.NewPostgres(pool)
			sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			bonus := reaction.NewReactor(welcomebonus.NewReaction(bus), checkpointStore, sub, test.DiscardLogger())
			go func() { _ = bonus.Start(ctx) }()
			go func() {
				_ = projection.NewProjector(balance.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(ctx)
			}()
			require.Eventually(t, func() bool {
				cp, err := checkpointStore.Get(ctx, bonus.Name())
				return err == nil && cp != nil
			}, 5*time.Second, 50*time.Millisecond, "the welcome bonus reaction did not save its start")

			id := uuid.New()
			require.NoError(t, bus.Dispatch(ctx, domain.NewCommand(accountCommand.Open{AccountID: id, Owner: "alice"})))
			require.NoError(t, bus.Dispatch(ctx, domain.NewCommand(accountCommand.Deposit{AccountID: id, Amount: 500})))
			require.NoError(t, bus.Dispatch(ctx, domain.NewCommand(accountCommand.Withdraw{AccountID: id, Amount: 200})))

			query := balance.NewQuery(pool)
			require.Eventually(t, func() bool {
				b, err := query.ByID(ctx, id.String())
				return err == nil && b != nil && b.Balance == 500-200+welcomebonus.Amount
			}, 5*time.Second, 50*time.Millisecond)

			require.NoError(t, bus.Dispatch(ctx, domain.NewCommand(accountCommand.Close{AccountID: id})))

			require.Eventually(t, func() bool {
				b, err := query.ByID(ctx, id.String())
				return err == nil && b != nil && b.Status == "closed" && b.Balance == 500-200+welcomebonus.Amount
			}, 5*time.Second, 50*time.Millisecond)
		})
	})
}
