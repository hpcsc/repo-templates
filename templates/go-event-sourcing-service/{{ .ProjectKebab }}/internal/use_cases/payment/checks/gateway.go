package checks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	paymentEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/outside"
)

var _ reaction.Interface = (*Gateway)(nil)

func NewGateway(payments *payment.Repository, client *outside.Client, cfg outside.Config, logger *slog.Logger) *Gateway {
	return &Gateway{
		payments: payments,
		client:   client,
		cfg:      cfg,
		logger:   logger,
	}
}

type Gateway struct {
	payments *payment.Repository
	client   *outside.Client
	cfg      outside.Config
	logger   *slog.Logger
}

type CheckRequest struct {
	PaymentID    uuid.UUID `json:"paymentId"`
	Amount       int64     `json:"amount"`
	CreditorIBAN string    `json:"creditorIban"`
	WebhookURL   string    `json:"webhookUrl"`
}

func (g *Gateway) Name() string {
	return "payment-checks"
}

func (g *Gateway) EventTypes() []string {
	return []string{
		paymentEvent.PaymentFunded{}.EventType(),
		paymentEvent.PaymentChecksRetried{}.EventType(),
	}
}

// Handle logs a failed request and does not return it, because a returned error stops the consumer,
// and then no payment gets its checks. The deadline of the checks asks again.
func (g *Gateway) Handle(ctx context.Context, evt *domain.Event) error {
	var paymentID uuid.UUID
	var attempt int
	switch payload := evt.Payload.(type) {
	case *paymentEvent.PaymentFunded:
		paymentID, attempt = payload.PaymentID, 1
	case *paymentEvent.PaymentChecksRetried:
		paymentID, attempt = payload.PaymentID, payload.Attempt
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}

	requested, err := g.payments.Requested(ctx, paymentID)
	if err != nil {
		return err
	}

	err = errors.Join(
		g.ask(ctx, g.cfg.FraudServiceURL+"/checks", "fraud-check", attempt, "/fraud-checks", requested),
		g.ask(ctx, g.cfg.SanctionsServiceURL+"/screenings", "sanctions-screening", attempt, "/sanctions-screenings", requested),
	)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		g.logger.Error(fmt.Sprintf("%s failed to ask for the checks of payment %s: %v", g.Name(), paymentID, err))
	}
	return nil
}

// ask gives each attempt its own key, so a check service that lost a request runs the check again.
func (g *Gateway) ask(ctx context.Context, url string, call string, attempt int, webhookPath string, requested *paymentEvent.PaymentRequested) error {
	return g.client.Post(ctx, url, outside.KeyFor(requested.PaymentID, fmt.Sprintf("%s/%d", call, attempt)), CheckRequest{
		PaymentID:    requested.PaymentID,
		Amount:       requested.Amount,
		CreditorIBAN: requested.CreditorIBAN,
		WebhookURL:   g.cfg.WebhookURL + webhookPath,
	}, nil)
}
