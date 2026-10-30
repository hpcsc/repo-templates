//go:build integration

package payment_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api"
	apiUseCases "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases"
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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/fakeservices"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/balance"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/command"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/checks"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/funding"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/holds"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/outside"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/submission"
	"github.com/stretchr/testify/require"
)

func TestPaymentProcess(t *testing.T) {
	t.Run("funding", func(t *testing.T) {
		t.Run("fails the payment when the account cannot hold the amount", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 100)

			paymentID := sys.requestPayment(t, accountID, 300, "DE89370400440532013000")

			sys.waitFor(t, payment.StreamID(paymentID), paymentEvent.PaymentFailed{}.EventType())
			require.Contains(t, sys.payloads(t, payment.StreamID(paymentID)), any(&paymentEvent.PaymentFailed{PaymentID: paymentID, AccountID: accountID, Reason: account.ErrInsufficientFunds.Error()}))
		})
	})

	t.Run("end", func(t *testing.T) {
		t.Run("settles the payment, captures the hold, and takes the amount off the balance", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 500)

			paymentID := sys.requestPayment(t, accountID, 300, "DE89370400440532013000")

			sys.waitFor(t, account.StreamID(accountID), accountEvent.HoldCaptured{}.EventType())
			require.Contains(t, sys.payloads(t, payment.StreamID(paymentID)), any(&paymentEvent.PaymentSettled{PaymentID: paymentID, AccountID: accountID}))
			require.Contains(t, sys.payloads(t, account.StreamID(accountID)), any(&accountEvent.HoldCaptured{AccountID: accountID, PaymentID: paymentID, Amount: 300}))
			require.Eventually(t, func() bool {
				b, err := sys.balances.ByID(context.Background(), accountID.String())
				return err == nil && b != nil && b.Balance == 500-300
			}, 10*time.Second, 50*time.Millisecond, "the balance does not show the captured amount")
		})

		t.Run("refuses a payment that fails the fraud check, and releases the hold", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 20000)

			paymentID := sys.requestPayment(t, accountID, 15000, "DE89370400440532013000")

			sys.waitFor(t, account.StreamID(accountID), accountEvent.HoldReleased{}.EventType())
			require.Contains(t, sys.payloads(t, payment.StreamID(paymentID)), any(&paymentEvent.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "amount above the limit of the fraud check"}))
		})

		t.Run("releases the hold of a payment that the other bank returns", func(t *testing.T) {
			sys := startSystem(t)
			accountID := sys.openAccount(t, 500)

			paymentID := sys.requestPayment(t, accountID, 300, "DE89370400440532012222")

			sys.waitFor(t, account.StreamID(accountID), accountEvent.HoldReleased{}.EventType())
			require.Contains(t, sys.payloads(t, payment.StreamID(paymentID)), any(&paymentEvent.PaymentReturned{PaymentID: paymentID, AccountID: accountID, Reason: "account closed at the other bank"}))
		})
	})
}

type system struct {
	bus      *command.Bus
	stream   event.Stream
	balances *balance.Query
}

// startSystem runs the consumers of the payment, the webhook routes of the API, and the fake outside
// systems, so a payment goes through the same steps as in `task run`.
func startSystem(t *testing.T) *system {
	pool := test.NewDBPool(t)
	reg := registry.New()
	use_cases.RegisterEvents(reg)
	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus(command.WithHistory(eventStream))
	use_cases.RegisterHandlers(bus, eventStream)

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("valid-token"), 0o600))
	router := chi.NewRouter()
	require.NoError(t, apiUseCases.Register(router, &api.Config{TokenPath: tokenPath}, pool, test.DiscardLogger()))
	apiServer := httptest.NewServer(router)
	t.Cleanup(apiServer.Close)
	fakes := httptest.NewServer(fakeservices.NewHandler("valid-token", 10*time.Millisecond, test.DiscardLogger()))
	t.Cleanup(fakes.Close)
	outsideCfg := outside.Config{
		FraudServiceURL:     fakes.URL + "/fraud",
		SanctionsServiceURL: fakes.URL + "/sanctions",
		PaymentNetworkURL:   fakes.URL + "/network",
		WebhookURL:          apiServer.URL + "/webhooks",
	}

	payments := payment.NewRepository(eventStream)
	client := outside.NewClient(outside.WithRetries(3, 10*time.Millisecond))
	checkpointStore := store.NewPostgres(pool)
	sub := subscription.NewPostgres(pool, reg, test.DiscardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, c := range []interface{ Start(context.Context) error }{
		projection.NewProjector(balance.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()),
		reaction.NewReactor(funding.NewReaction(bus), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
		reaction.NewReactor(holds.NewReaction(bus), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
		reaction.NewReactor(checks.NewGateway(payments, client, outsideCfg, test.DiscardLogger()), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
		reaction.NewReactor(submission.NewGateway(bus, payments, client, outsideCfg, test.DiscardLogger()), checkpointStore, sub, test.DiscardLogger(), reaction.StartAtBeginning()),
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

func (s *system) requestPayment(t *testing.T, accountID uuid.UUID, amount int64, creditorIBAN string) uuid.UUID {
	id := uuid.New()
	s.dispatch(t, paymentCommand.Request{PaymentID: id, AccountID: accountID, CreditorIBAN: creditorIBAN, Amount: amount})
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
