package process

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Status struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{
		pool: pool,
	}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) ByID(ctx context.Context, id string) (*Status, error) {
	var s Status
	err := q.pool.QueryRow(ctx,
		"SELECT id::text, from_account::text, to_account::text, amount, state, reason FROM transfer_processes WHERE id = $1",
		id,
	).Scan(&s.ID, &s.From, &s.To, &s.Amount, &s.State, &s.Reason)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read transfer %s: %w", id, err)
	}
	return &s, nil
}
