package subscription

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/row"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultBatchSize    = 100
	DefaultPollInterval = 5 * time.Second

	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

// the transaction ID filter stops a read at the oldest open transaction, so a late commit cannot add an event below the position
const readQuery = `SELECT ` + row.Columns + `
FROM events
WHERE (transaction_id, sequence) > ($1, $2)
  AND transaction_id < pg_snapshot_xmin(pg_current_snapshot())
  AND (cardinality($3::text[]) = 0 OR type = ANY($3))
ORDER BY transaction_id, sequence
LIMIT $4`

const lastPositionQuery = `SELECT transaction_id, sequence
FROM events
WHERE transaction_id < pg_snapshot_xmin(pg_current_snapshot())
ORDER BY transaction_id DESC, sequence DESC
LIMIT 1`

var _ event.Subscription = (*postgres)(nil)

type PostgresOption func(*postgres)

func WithBatchSize(batchSize int) PostgresOption {
	return func(p *postgres) {
		p.batchSize = batchSize
	}
}

func WithPollInterval(interval time.Duration) PostgresOption {
	return func(p *postgres) {
		p.pollInterval = interval
	}
}

func NewPostgres(pool *pgxpool.Pool, reg *registry.OfEvents, logger *slog.Logger, opts ...PostgresOption) event.Subscription {
	p := &postgres{
		pool:         pool,
		registry:     reg,
		logger:       logger,
		batchSize:    DefaultBatchSize,
		pollInterval: DefaultPollInterval,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type postgres struct {
	pool         *pgxpool.Pool
	registry     *registry.OfEvents
	logger       *slog.Logger
	batchSize    int
	pollInterval time.Duration
}

func (p *postgres) SubscribeToAll(ctx context.Context, after *domain.Position, eventTypes []string) (<-chan domain.Event, error) {
	events := make(chan domain.Event, p.batchSize)
	signal := make(chan struct{}, 1)

	go p.listen(ctx, signal)
	go p.read(ctx, after, eventTypes, signal, events)

	return events, nil
}

func (p *postgres) LastPosition(ctx context.Context) (*domain.Position, error) {
	var position domain.Position
	err := p.pool.QueryRow(ctx, lastPositionQuery).Scan(&position.TransactionID, &position.Sequence)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the last position: %w", err)
	}
	return &position, nil
}

func (p *postgres) read(ctx context.Context, after *domain.Position, eventTypes []string, signal <-chan struct{}, events chan<- domain.Event) {
	defer close(events)

	var position domain.Position
	if after != nil {
		position = *after
	}
	if eventTypes == nil {
		eventTypes = []string{}
	}

	attempt := 0
	for {
		stored, err := p.readBatch(ctx, position, eventTypes)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			attempt++
			p.logger.Warn("failed to read events", "error", err, "attempt", attempt)
			if !wait(ctx, calculateBackoff(attempt), nil) {
				return
			}
			continue
		}
		attempt = 0

		for _, e := range stored {
			evt, err := e.ToDomain(p.registry)
			if errors.Is(err, registry.ErrUnknownEventType) {
				p.logger.Warn("skipped event of unknown type", "type", e.Type, "position", e.Position)
				position = e.Position
				continue
			}
			if err != nil {
				p.logger.Error("subscription stopped", "error", err, "afterPosition", position)
				return
			}

			select {
			case events <- evt:
				position = evt.Position
			case <-ctx.Done():
				return
			}
		}

		if len(stored) == p.batchSize {
			continue
		}

		if !wait(ctx, p.pollInterval, signal) {
			return
		}
	}
}

func (p *postgres) readBatch(ctx context.Context, after domain.Position, eventTypes []string) ([]row.Event, error) {
	rows, err := p.pool.Query(ctx, readQuery, after.TransactionID, after.Sequence, eventTypes, p.batchSize)
	if err != nil {
		return nil, err
	}
	return row.Scan(rows)
}

func (p *postgres) listen(ctx context.Context, signal chan<- struct{}) {
	attempt := 0
	for {
		listened, err := p.listenOnce(ctx, signal)
		if ctx.Err() != nil {
			return
		}
		if listened {
			attempt = 0
		}
		attempt++
		p.logger.Warn("event listener stopped", "error", err, "attempt", attempt)
		if !wait(ctx, calculateBackoff(attempt), nil) {
			return
		}
	}
}

func (p *postgres) listenOnce(ctx context.Context, signal chan<- struct{}) (bool, error) {
	conn, err := pgx.ConnectConfig(ctx, p.pool.Config().ConnConfig.Copy())
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{event.NotifyChannel}.Sanitize()); err != nil {
		return false, err
	}

	// events can arrive while the listener is down, so each new listener asks for one read
	notify(signal)

	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return true, err
		}
		notify(signal)
	}
}

func notify(signal chan<- struct{}) {
	select {
	case signal <- struct{}{}:
	default:
	}
}

func wait(ctx context.Context, timeout time.Duration, signal <-chan struct{}) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-signal:
		return true
	case <-timer.C:
		return true
	}
}

func calculateBackoff(attempt int) time.Duration {
	backoff := initialBackoff
	for i := 0; i < attempt && backoff < maxBackoff; i++ {
		backoff *= 2
	}
	return min(backoff, maxBackoff)
}
