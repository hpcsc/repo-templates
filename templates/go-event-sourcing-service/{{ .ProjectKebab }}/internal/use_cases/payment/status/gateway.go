package status

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

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

type StatusAnswer struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (g *Gateway) Name() string {
	return "payment-status"
}

func (g *Gateway) EventTypes() []string {
	return []string{paymentEvent.PaymentStatusQueried{}.EventType()}
}

// Handle records a settlement or a return, and records nothing while the network has no answer: the
// next deadline of the settlement asks again. It logs a failed request, because a returned error
// stops the consumer.
func (g *Gateway) Handle(ctx context.Context, evt *domain.Event) error {
	queried, ok := evt.Payload.(*paymentEvent.PaymentStatusQueried)
	if !ok {
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}

	submitted, err := g.payments.Submitted(ctx, queried.PaymentID)
	if err != nil {
		return err
	}

	var answer StatusAnswer
	err = g.client.Get(ctx, g.cfg.PaymentNetworkURL+"/payments/"+url.PathEscape(submitted.NetworkReference), &answer)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		g.logger.Error(fmt.Sprintf("%s failed to ask for the status of payment %s: %v", g.Name(), queried.PaymentID, err))
		return nil
	}

	switch answer.Status {
	case "settled":
		return g.bus.Dispatch(ctx, dispatch.CommandFor(g.Name(), evt, paymentCommand.RecordSettlement{PaymentID: queried.PaymentID}))
	case "returned":
		return g.bus.Dispatch(ctx, dispatch.CommandFor(g.Name(), evt, paymentCommand.RecordReturn{PaymentID: queried.PaymentID, Reason: answer.Reason}))
	default:
		return nil
	}
}
