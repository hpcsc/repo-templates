package summary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/jackc/pgx/v5"
)

var _ projection.Interface = (*Projection)(nil)

func NewProjection() *Projection {
	return &Projection{}
}

type Projection struct{}

func (p *Projection) Name() string {
	return "payments"
}

func (p *Projection) EventTypes() []string {
	return []string{
		event.PaymentRequested{}.EventType(),
		event.PaymentFunded{}.EventType(),
		event.PaymentFailed{}.EventType(),
		event.PaymentCleared{}.EventType(),
		event.PaymentRefused{}.EventType(),
		event.PaymentSubmitted{}.EventType(),
		event.PaymentEscalated{}.EventType(),
		event.PaymentSettled{}.EventType(),
		event.PaymentReturned{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.PaymentRequested:
		_, err := tx.Exec(ctx,
			`INSERT INTO payments (id, account_id, creditor_iban, amount, status, version, updated_at) VALUES ($1, $2, $3, $4, 'requested', $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			payload.PaymentID, payload.AccountID, payload.CreditorIBAN, payload.Amount, evt.Version, evt.RecordedAt,
		)
		return err
	case *event.PaymentFunded:
		return p.move(ctx, tx, payload.PaymentID, "funded", "", evt)
	case *event.PaymentFailed:
		return p.move(ctx, tx, payload.PaymentID, "failed", payload.Reason, evt)
	case *event.PaymentCleared:
		return p.move(ctx, tx, payload.PaymentID, "cleared", "", evt)
	case *event.PaymentRefused:
		return p.move(ctx, tx, payload.PaymentID, "refused", payload.Reason, evt)
	case *event.PaymentSubmitted:
		_, err := tx.Exec(ctx,
			`UPDATE payments SET status = 'submitted', network_reference = $2, version = $3, updated_at = $4
			WHERE id = $1 AND version < $3`,
			payload.PaymentID, payload.NetworkReference, evt.Version, evt.RecordedAt,
		)
		return err
	case *event.PaymentEscalated:
		return p.move(ctx, tx, payload.PaymentID, "escalated", payload.Reason, evt)
	case *event.PaymentSettled:
		return p.move(ctx, tx, payload.PaymentID, "settled", "", evt)
	case *event.PaymentReturned:
		return p.move(ctx, tx, payload.PaymentID, "returned", payload.Reason, evt)
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}

func (p *Projection) move(ctx context.Context, tx pgx.Tx, paymentID uuid.UUID, status string, reason string, evt *domain.Event) error {
	_, err := tx.Exec(ctx,
		`UPDATE payments SET status = $2, reason = $3, version = $4, updated_at = $5
		WHERE id = $1 AND version < $4`,
		paymentID, status, reason, evt.Version, evt.RecordedAt,
	)
	return err
}
