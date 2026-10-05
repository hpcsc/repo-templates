package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ checkpoint.Store = (*postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) checkpoint.Store {
	return &postgres{
		pool: pool,
	}
}

type postgres struct {
	pool *pgxpool.Pool
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (p *postgres) Get(ctx context.Context, name string) (*domain.Position, error) {
	var position domain.Position
	err := p.pool.QueryRow(ctx,
		"SELECT transaction_id, sequence FROM checkpoints WHERE name = $1",
		name,
	).Scan(&position.TransactionID, &position.Sequence)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get the checkpoint of %s: %w", name, err)
	}
	return &position, nil
}

func (p *postgres) Set(ctx context.Context, name string, position domain.Position) error {
	return p.set(ctx, p.pool, name, position)
}

func (p *postgres) SetInTx(ctx context.Context, tx pgx.Tx, name string, position domain.Position) error {
	return p.set(ctx, tx, name, position)
}

func (p *postgres) set(ctx context.Context, db executor, name string, position domain.Position) error {
	_, err := db.Exec(ctx,
		`INSERT INTO checkpoints (name, transaction_id, sequence)
		VALUES ($1, $2, $3)
		ON CONFLICT (name) DO UPDATE SET transaction_id = $2, sequence = $3, updated_at = NOW()`,
		name,
		position.TransactionID,
		position.Sequence,
	)
	if err != nil {
		return fmt.Errorf("failed to set the checkpoint of %s: %w", name, err)
	}
	return nil
}
