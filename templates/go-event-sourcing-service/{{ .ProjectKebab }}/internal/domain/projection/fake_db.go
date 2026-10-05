package projection

import (
	"context"

	"github.com/jackc/pgx/v5"
)

var _ DB = (*FakeDB)(nil)

func NewFakeDB() *FakeDB {
	return &FakeDB{}
}

type FakeDB struct {
	Committed  int
	RolledBack int
}

func (f *FakeDB) Begin(_ context.Context) (pgx.Tx, error) {
	return &fakeTx{db: f}, nil
}

type fakeTx struct {
	pgx.Tx
	db     *FakeDB
	closed bool
}

func (t *fakeTx) Commit(_ context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	t.closed = true
	t.db.Committed++
	return nil
}

func (t *fakeTx) Rollback(_ context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	t.closed = true
	t.db.RolledBack++
	return nil
}
