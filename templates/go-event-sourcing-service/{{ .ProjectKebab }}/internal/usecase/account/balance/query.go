package balance

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Balance struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Balance int64  `json:"balance"`
	Status  string `json:"status"`
	Version uint64 `json:"version"`
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{
		pool: pool,
	}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) ByID(ctx context.Context, id string) (*Balance, error) {
	var b Balance
	err := q.pool.QueryRow(ctx,
		"SELECT id::text, owner, balance, status, version FROM account_balances WHERE id = $1",
		id,
	).Scan(&b.ID, &b.Owner, &b.Balance, &b.Status, &b.Version)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the balance of account %s: %w", id, err)
	}
	return &b, nil
}
