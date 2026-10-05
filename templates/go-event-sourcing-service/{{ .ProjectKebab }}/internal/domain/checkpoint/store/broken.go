package store

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/jackc/pgx/v5"
)

var _ checkpoint.Store = (*broken)(nil)

func NewBroken() *broken {
	return &broken{}
}

type broken struct {
	getErr error
	setErr error
}

func (b *broken) WithGetError(err error) *broken {
	b.getErr = err
	return b
}

func (b *broken) WithSetError(err error) *broken {
	b.setErr = err
	return b
}

func (b *broken) Get(context.Context, string) (*domain.Position, error) {
	return nil, b.getErr
}

func (b *broken) Set(context.Context, string, domain.Position) error {
	return b.setErr
}

func (b *broken) SetInTx(context.Context, pgx.Tx, string, domain.Position) error {
	return b.setErr
}
