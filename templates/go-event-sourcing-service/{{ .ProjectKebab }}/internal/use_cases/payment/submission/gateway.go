package submission

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/dispatch"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/command"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/outside"
)

var _ reaction.Interface = (*Gateway)(nil)

func NewGateway(bus *command.Bus, payments *payment.Repository, client *outside.Client, cfg outside.Config, logger *slog.Logger) *Gateway {
	return &Gateway{
		bus:      bus,
		payments: payments,
		client:   client,
		cfg:      cfg,
		logger:   logger,
	}
}

type Gateway struct {
	bus      *command.Bus
	payments *payment.Repository
	client   *outside.Client
	cfg      outside.Config
	logger   *slog.Logger
}

type SubmitRequest struct {
	PaymentID    uuid.UUID `json:"paymentId"`
	Amount       int64     `json:"amount"`
	CreditorIBAN string    `json:"creditorIban"`
	WebhookURL   string    `json:"webhookUrl"`
}

type SubmitAnswer struct {
	Reference string `json:"reference"`
}

func (g *Gateway) Name() string {
	return "payment-submission"
}

func (g *Gateway) EventTypes() []string {
	return []string{paymentEvent.PaymentCleared{}.EventType()}
}

// Handle logs a failed request and does not return it, because a returned error stops the consumer,
// and then no payment goes to the network.
func (g *Gateway) Handle(ctx context.Context, evt *domain.Event) error {
	cleared, ok := evt.Payload.(*paymentEvent.PaymentCleared)
	if !ok {
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}

	requested, err := g.payments.Requested(ctx, cleared.PaymentID)
	if err != nil {
		return err
	}

	var answer SubmitAnswer
	err = g.client.Post(ctx, g.cfg.PaymentNetworkURL+"/payments", outside.KeyFor(cleared.PaymentID, "submission", 1), SubmitRequest{
		PaymentID:    requested.PaymentID,
		Amount:       requested.Amount,
		CreditorIBAN: requested.CreditorIBAN,
		WebhookURL:   g.cfg.WebhookURL + "/payment-network",
	}, &answer)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		g.logger.Error(fmt.Sprintf("%s failed to send payment %s: %v", g.Name(), cleared.PaymentID, err))
		return nil
	}

	return g.bus.Dispatch(ctx, dispatch.CommandFor(g.Name(), evt, paymentCommand.RecordSubmission{
		PaymentID:        cleared.PaymentID,
		NetworkReference: answer.Reference,
	}))
}
