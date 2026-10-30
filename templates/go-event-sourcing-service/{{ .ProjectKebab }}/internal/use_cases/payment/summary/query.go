package summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Summary struct {
	ID               string `json:"id"`
	AccountID        string `json:"accountId"`
	CreditorIBAN     string `json:"creditorIban"`
	Amount           int64  `json:"amount"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	NetworkReference string `json:"networkReference"`
	Version          uint64 `json:"version"`
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{
		pool: pool,
	}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) ByID(ctx context.Context, id string) (*Summary, error) {
	var s Summary
	err := q.pool.QueryRow(ctx,
		`SELECT id::text, account_id::text, creditor_iban, amount, status, reason, network_reference, version
		FROM payments WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.AccountID, &s.CreditorIBAN, &s.Amount, &s.Status, &s.Reason, &s.NetworkReference, &s.Version)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read payment %s: %w", id, err)
	}
	return &s, nil
}
