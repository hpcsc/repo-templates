package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint"
	"github.com/jackc/pgx/v5"
)

var _ checkpoint.Store = (*memory)(nil)

func NewEmptyMemory() checkpoint.Store {
	return NewMemory(make(map[string]domain.Position))
}

func NewMemory(positionByName map[string]domain.Position) checkpoint.Store {
	return &memory{
		positionByName: positionByName,
	}
}

type memory struct {
	positionByName map[string]domain.Position
	mutex          sync.RWMutex
}

func (m *memory) Get(_ context.Context, name string) (*domain.Position, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	position, exist := m.positionByName[name]
	if !exist {
		return nil, nil
	}
	return &position, nil
}

func (m *memory) Set(_ context.Context, name string, position domain.Position) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if saved, exist := m.positionByName[name]; exist && position.Before(saved) {
		return fmt.Errorf("%w: %s, position %+v", checkpoint.ErrMovedBack, name, position)
	}
	m.positionByName[name] = position
	return nil
}

func (m *memory) SetInTx(ctx context.Context, _ pgx.Tx, name string, position domain.Position) error {
	return m.Set(ctx, name, position)
}
