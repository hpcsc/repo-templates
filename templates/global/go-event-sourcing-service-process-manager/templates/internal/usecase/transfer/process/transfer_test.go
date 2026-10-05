//go:build integration

package process_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/common/test"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/checkpoint/store"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/registry"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/stream"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/subscription"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/projection"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/reaction"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/account"
	accountCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/account/command"
	accountEvent "github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/account/event"
	transferCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/transfer/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/transfer/process"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestTransferProcess(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		t.Run("moves the money when both accounts accept", func(t *testing.T) {
			sys := startSystem(t)
			from, to := sys.openAccount(t, 500), sys.openAccount(t, 0)

			id := sys.startTransfer(t, from, to, 300)

			sys.requireState(t, id, string(process.Completed), "")
			require.Contains(t, sys.payloads(t, from), any(&accountEvent.TransferDebited{AccountID: from, TransferID: id, Amount: 300}))
			require.Contains(t, sys.payloads(t, to), any(&accountEvent.TransferCredited{AccountID: to, TransferID: id, Amount: 300}))
		})

		t.Run("fails with nothing to undo when the source has too little money", func(t *testing.T) {
			sys := startSystem(t)
			from, to := sys.openAccount(t, 100), sys.openAccount(t, 0)

			id := sys.startTransfer(t, from, to, 5000)

			sys.requireState(t, id, string(process.Failed), account.ErrInsufficientFunds.Error())
			require.Contains(t, sys.payloads(t, from), any(&accountEvent.TransferDebitRejected{AccountID: from, TransferID: id, Amount: 5000, Reason: account.ErrInsufficientFunds.Error()}))
			require.Len(t, sys.payloads(t, to), 1)
		})

		t.Run("refunds the source when the target account is closed", func(t *testing.T) {
			sys := startSystem(t)
			from, to := sys.openAccount(t, 500), sys.openAccount(t, 0)
			require.NoError(t, sys.bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Close{AccountID: to})))

			id := sys.startTransfer(t, from, to, 300)

			sys.requireState(t, id, string(process.Failed), account.ErrClosed.Error())
			fromPayloads := sys.payloads(t, from)
			require.Contains(t, fromPayloads, any(&accountEvent.TransferDebited{AccountID: from, TransferID: id, Amount: 300}))
			require.Contains(t, fromPayloads, any(&accountEvent.TransferRefunded{AccountID: from, TransferID: id, Amount: 300}))
			require.Contains(t, sys.payloads(t, to), any(&accountEvent.TransferCreditRejected{AccountID: to, TransferID: id, Amount: 300, Reason: account.ErrClosed.Error()}))
		})
	})
}

type system struct {
	bus      *command.Bus
	stream   event.Stream
	statuses *process.Query
}

func startSystem(t *testing.T) *system {
	pool := test.NewDBPool(t)
	reg := registry.New()
	usecase.RegisterEvents(reg)
	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus()
	usecase.RegisterHandlers(bus, eventStream)

	startConsumers(t, pool, reg, eventStream, bus)

	return &system{bus: bus, stream: eventStream, statuses: process.NewQuery(pool)}
}

func startConsumers(t *testing.T, pool *pgxpool.Pool, reg *registry.OfEvents, eventStream event.Stream, bus *command.Bus) {
	checkpointStore := store.NewPostgres(pool)
	sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for _, c := range []interface{ Start(context.Context) error }{
		projection.NewProjector(process.NewProjection(eventStream), pool, checkpointStore, sub, test.DiscardLogger()),
		reaction.NewReactor(process.NewDispatcher(bus), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Start(ctx)
		}()
	}

	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
}

func (s *system) openAccount(t *testing.T, deposit int64) uuid.UUID {
	id := uuid.New()
	require.NoError(t, s.bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Open{AccountID: id, Owner: "owner-" + id.String()})))
	if deposit > 0 {
		require.NoError(t, s.bus.Dispatch(context.Background(), domain.NewCommand(accountCommand.Deposit{AccountID: id, Amount: deposit})))
	}
	return id
}

func (s *system) startTransfer(t *testing.T, from uuid.UUID, to uuid.UUID, amount int64) uuid.UUID {
	id := uuid.New()
	require.NoError(t, s.bus.Dispatch(context.Background(), domain.NewCommand(transferCommand.Start{TransferID: id, From: from, To: to, Amount: amount})))
	return id
}

func (s *system) requireState(t *testing.T, id uuid.UUID, state string, reason string) {
	require.Eventually(t, func() bool {
		status, err := s.statuses.ByID(context.Background(), id.String())
		return err == nil && status != nil && status.State == state && status.Reason == reason
	}, 10*time.Second, 50*time.Millisecond, "transfer %s did not reach state %s", id, state)
}

func (s *system) payloads(t *testing.T, accountID uuid.UUID) []any {
	events, err := s.stream.EventsForStream(context.Background(), account.StreamID(accountID))
	require.NoError(t, err)
	result := make([]any, 0, len(events))
	for _, e := range events {
		result = append(result, e.Payload)
	}
	return result
}
