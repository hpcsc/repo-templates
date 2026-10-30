package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultInterval = time.Second

// A ping that ends on its timeout closes the connection, and the session then gives up the lock.
// So the timeout allows for a slow database, and only a lost connection ends the lock.
const pingTimeout = 5 * time.Second

var ErrLost = errors.New("the lock ended")

type Option func(*Postgres)

func WithInterval(interval time.Duration) Option {
	return func(p *Postgres) {
		p.interval = interval
	}
}

func NewPostgres(pool *pgxpool.Pool, opts ...Option) *Postgres {
	p := &Postgres{
		pool:     pool,
		interval: DefaultInterval,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type Postgres struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

func (p *Postgres) Hold(ctx context.Context, name string, start func(context.Context) error) error {
	held, release, err := p.acquire(ctx, name)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	defer release()

	err = start(held)
	if ctx.Err() == nil && held.Err() != nil {
		return context.Cause(held)
	}
	return err
}

func (p *Postgres) acquire(ctx context.Context, name string) (context.Context, func(), error) {
	config := p.pool.Config().ConnConfig.Copy()
	config.RuntimeParams["application_name"] = "lock " + name

	// a pooled connection goes back to the pool and keeps the lock of its session
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect for the lock of %s: %w", name, err)
	}

	if err := p.waitFor(ctx, conn, name); err != nil {
		_ = conn.Close(context.Background())
		return nil, nil, err
	}

	held, end := context.WithCancelCause(ctx)
	checked := make(chan struct{})
	go func() {
		defer close(checked)
		p.check(held, end, conn, name)
	}()

	release := func() {
		end(nil)
		<-checked
		_ = conn.Close(context.Background())
	}
	return held, release, nil
}

func (p *Postgres) waitFor(ctx context.Context, conn *pgx.Conn, name string) error {
	for {
		var locked bool
		if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended('lock/' || $1, 0))", name).Scan(&locked); err != nil {
			return fmt.Errorf("failed to take the lock of %s: %w", name, err)
		}
		if locked {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.interval):
		}
	}
}

func (p *Postgres) check(held context.Context, end context.CancelCauseFunc, conn *pgx.Conn, name string) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-held.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(held, pingTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil && held.Err() == nil {
				end(fmt.Errorf("%w: %s: %w", ErrLost, name, err))
				return
			}
		}
	}
}
