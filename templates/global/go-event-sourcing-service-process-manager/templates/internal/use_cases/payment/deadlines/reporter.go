package deadlines

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/command"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/scheduler"
	paymentCommand "github.com/hpcsc/go-event-sourcing-service-project/internal/use_cases/payment/command"
	"github.com/jackc/pgx/v5/pgxpool"
)

const batchSize = 100

var _ scheduler.Interface = (*Reporter)(nil)

func NewReporter(pool *pgxpool.Pool, bus *command.Bus) *Reporter {
	return &Reporter{
		pool: pool,
		bus:  bus,
	}
}

// Reporter tells each payment that a deadline passed. The payment decides what that means.
type Reporter struct {
	pool *pgxpool.Pool
	bus  *command.Bus
}

type due struct {
	paymentID uuid.UUID
	stage     string
	attempt   int
}

func (r *Reporter) Name() string {
	return "payment-timeouts"
}

func (r *Reporter) Tick(ctx context.Context, now time.Time) error {
	items, err := r.due(ctx, now)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := r.report(ctx, item); err != nil {
			return fmt.Errorf("failed to report the passed deadline of payment %s: %w", item.paymentID, err)
		}
	}
	return nil
}

func (r *Reporter) due(ctx context.Context, now time.Time) ([]due, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT payment_id, stage, attempt FROM payment_deadlines WHERE deadline <= $1 ORDER BY deadline LIMIT $2`,
		now, batchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read the passed deadlines: %w", err)
	}
	defer rows.Close()

	var items []due
	for rows.Next() {
		var item due
		if err := rows.Scan(&item.paymentID, &item.stage, &item.attempt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// report sends a command with an ID from the payment, the stage and the attempt, so a tick that runs
// again before the todo list changes sends no new command.
func (r *Reporter) report(ctx context.Context, item due) error {
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s/%s/%s/%d", r.Name(), item.paymentID, item.stage, item.attempt)))
	switch item.stage {
	case Checks:
		return r.bus.Dispatch(ctx, domain.Command[paymentCommand.TimeOutChecks]{ID: id, Payload: paymentCommand.TimeOutChecks{PaymentID: item.paymentID, Attempt: item.attempt}})
	case Submission:
		return r.bus.Dispatch(ctx, domain.Command[paymentCommand.TimeOutSubmission]{ID: id, Payload: paymentCommand.TimeOutSubmission{PaymentID: item.paymentID, Attempt: item.attempt}})
	case Settlement:
		return r.bus.Dispatch(ctx, domain.Command[paymentCommand.TimeOutSettlement]{ID: id, Payload: paymentCommand.TimeOutSettlement{PaymentID: item.paymentID, Attempt: item.attempt}})
	default:
		return fmt.Errorf("received unknown stage %q", item.stage)
	}
}
