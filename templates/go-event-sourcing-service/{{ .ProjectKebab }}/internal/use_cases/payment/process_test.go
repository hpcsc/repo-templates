//go:build integration

package payment_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/balance"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/funding"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/holds"
	"github.com/stretchr/testify/require"
)

func TestPaymentProcess(t *testing.T) {
	t.Run("funding", func(t *testing.T) {
		t.Run("funds the payment when the account holds the amount", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 500)

			paymentID := sys.requestPayment(t, accountID, 300)

			sys.waitFor(t, payment.StreamID(paymentID), paymentEvent.PaymentFunded{}.EventType())
			require.Contains(t, sys.payloads(t, account.StreamID(accountID)), any(&accountEvent.HoldPlaced{AccountID: accountID, PaymentID: paymentID, Amount: 300}))
		})

		t.Run("fails the payment when the account cannot hold the amount", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 100)

			paymentID := sys.requestPayment(t, accountID, 300)

			sys.waitFor(t, payment.StreamID(paymentID), paymentEvent.PaymentFailed{}.EventType())
			require.Contains(t, sys.payloads(t, payment.StreamID(paymentID)), any(&paymentEvent.PaymentFailed{PaymentID: paymentID, AccountID: accountID, Reason: account.ErrInsufficientFunds.Error()}))
		})
	})

	t.Run("end", func(t *testing.T) {
		t.Run("captures the hold and takes the amount off the balance when the payment settles", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 500)
			paymentID := sys.requestPayment(t, accountID, 300)
			sys.waitFor(t, payment.StreamID(paymentID), paymentEvent.PaymentFunded{}.EventType())

			sys.dispatch(t, paymentCommand.RecordFraudResult{PaymentID: paymentID, Passed: true})
			sys.dispatch(t, paymentCommand.RecordScreeningResult{PaymentID: paymentID, Passed: true})
			sys.dispatch(t, paymentCommand.RecordSettlement{PaymentID: paymentID})

			sys.waitFor(t, account.StreamID(accountID), accountEvent.HoldCaptured{}.EventType())
			require.Contains(t, sys.payloads(t, account.StreamID(accountID)), any(&accountEvent.HoldCaptured{AccountID: accountID, PaymentID: paymentID, Amount: 300}))
			require.Eventually(t, func() bool {
				b, err := sys.balances.ByID(context.Background(), accountID.String())
				return err == nil && b != nil && b.Balance == 500-300
			}, 10*time.Second, 50*time.Millisecond, "the balance does not show the captured amount")
		})

		t.Run("releases the hold when a check fails", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 500)
			paymentID := sys.requestPayment(t, accountID, 300)
			sys.waitFor(t, payment.StreamID(paymentID), paymentEvent.PaymentFunded{}.EventType())

			sys.dispatch(t, paymentCommand.RecordFraudResult{PaymentID: paymentID, Passed: false, Reason: "amount above the limit"})

			sys.waitFor(t, account.StreamID(accountID), accountEvent.HoldReleased{}.EventType())
			require.Contains(t, sys.payloads(t, account.StreamID(accountID)), any(&accountEvent.HoldReleased{AccountID: accountID, PaymentID: paymentID, Amount: 300}))
		})
	})
}

type system struct {
	bus      *command.Bus
	stream   event.Stream
	balances *balance.Query
}

func startSystem(t *testing.T) *system {
	pool := test.NewDBPool(t)
	reg := registry.New()
	use_cases.RegisterEvents(reg)
	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus()
	use_cases.RegisterHandlers(bus, eventStream)

	checkpointStore := store.NewPostgres(pool)
	sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, c := range []interface{ Start(context.Context) error }{
		projection.NewProjector(balance.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()),
		reaction.NewReactor(funding.NewReaction(bus), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
		reaction.NewReactor(holds.NewReaction(bus), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
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

	return &system{bus: bus, stream: eventStream, balances: balance.NewQuery(pool)}
}

func (s *system) openAccount(t *testing.T, deposit int64) uuid.UUID {
	id := uuid.New()
	s.dispatch(t, accountCommand.Open{AccountID: id, Owner: "owner-" + id.String()})
	s.dispatch(t, accountCommand.Deposit{AccountID: id, Amount: deposit})
	return id
}

func (s *system) requestPayment(t *testing.T, accountID uuid.UUID, amount int64) uuid.UUID {
	id := uuid.New()
	s.dispatch(t, paymentCommand.Request{PaymentID: id, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: amount})
	return id
}

func (s *system) dispatch[P any](t *testing.T, payload P) {
	require.NoError(t, s.bus.Dispatch(context.Background(), domain.NewCommand(payload)))
}

func (s *system) waitFor(t *testing.T, streamID string, eventType string) {
	require.Eventually(t, func() bool {
		events, err := s.stream.EventsForStream(context.Background(), streamID)
		if err != nil {
			return false
		}
		for _, e := range events {
			if e.Payload.EventType() == eventType {
				return true
			}
		}
		return false
	}, 10*time.Second, 50*time.Millisecond, "stream %s has no %s", streamID, eventType)
}

func (s *system) payloads(t *testing.T, streamID string) []any {
	events, err := s.stream.EventsForStream(context.Background(), streamID)
	require.NoError(t, err)
	result := make([]any, 0, len(events))
	for _, e := range events {
		result = append(result, e.Payload)
	}
	return result
}
