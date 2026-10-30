package deadlines

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/projection"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/event"
	"github.com/jackc/pgx/v5"
)

const (
	Checks     = "checks"
	Submission = "submission"
	Settlement = "settlement"
)

var _ projection.Interface = (*Projection)(nil)

func NewProjection() *Projection {
	return &Projection{}
}

// Projection keeps the todo list of the payment-timeouts scheduler: each open payment, the stage that
// waits, the attempt, and its deadline.
type Projection struct{}

func (p *Projection) Name() string {
	return "payment-deadlines"
}

func (p *Projection) EventTypes() []string {
	return []string{
		event.PaymentFunded{}.EventType(),
		event.PaymentChecksRetried{}.EventType(),
		event.PaymentCleared{}.EventType(),
		event.PaymentSubmissionRetried{}.EventType(),
		event.PaymentSubmitted{}.EventType(),
		event.PaymentStatusQueried{}.EventType(),
		event.PaymentFailed{}.EventType(),
		event.PaymentRefused{}.EventType(),
		event.PaymentEscalated{}.EventType(),
		event.PaymentSettled{}.EventType(),
		event.PaymentReturned{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.PaymentFunded:
		return p.wait(ctx, tx, payload.PaymentID, Checks, 1, payload.ChecksDeadline, evt)
	case *event.PaymentChecksRetried:
		return p.wait(ctx, tx, payload.PaymentID, Checks, payload.Attempt, payload.ChecksDeadline, evt)
	case *event.PaymentCleared:
		return p.wait(ctx, tx, payload.PaymentID, Submission, 1, payload.SubmissionDeadline, evt)
	case *event.PaymentSubmissionRetried:
		return p.wait(ctx, tx, payload.PaymentID, Submission, payload.Attempt, payload.SubmissionDeadline, evt)
	case *event.PaymentSubmitted:
		return p.wait(ctx, tx, payload.PaymentID, Settlement, 1, payload.SettlementDeadline, evt)
	case *event.PaymentStatusQueried:
		return p.wait(ctx, tx, payload.PaymentID, Settlement, payload.Attempt, payload.SettlementDeadline, evt)
	case *event.PaymentFailed:
		return p.remove(ctx, tx, payload.PaymentID)
	case *event.PaymentRefused:
		return p.remove(ctx, tx, payload.PaymentID)
	case *event.PaymentEscalated:
		return p.remove(ctx, tx, payload.PaymentID)
	case *event.PaymentSettled:
		return p.remove(ctx, tx, payload.PaymentID)
	case *event.PaymentReturned:
		return p.remove(ctx, tx, payload.PaymentID)
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}

func (p *Projection) wait(ctx context.Context, tx pgx.Tx, paymentID uuid.UUID, stage string, attempt int, deadline time.Time, evt *domain.Event) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO payment_deadlines (payment_id, stage, attempt, deadline, version) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (payment_id) DO UPDATE SET stage = $2, attempt = $3, deadline = $4, version = $5
		WHERE payment_deadlines.version < $5`,
		paymentID, stage, attempt, deadline, evt.Version,
	)
	return err
}

func (p *Projection) remove(ctx context.Context, tx pgx.Tx, paymentID uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM payment_deadlines WHERE payment_id = $1`, paymentID)
	return err
}
