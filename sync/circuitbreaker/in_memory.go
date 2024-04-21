package circuitbreaker

import (
	"context"
	"sync"
)

type InMemory struct {
	rw   sync.RWMutex
	data map[string]*State
}

func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]*State),
	}
}

var _ store = (*InMemory)(nil)

func (i *InMemory) Get(ctx context.Context, key string) (*State, error) {
	i.rw.RLock()
	defer i.rw.RUnlock()

	res, ok := i.data[key]
	if !ok {
		// Null object.
		return &State{}, nil
	}

	cp := *res
	return &cp, nil
}

func (i *InMemory) Set(ctx context.Context, key string, res *State) error {
	i.rw.Lock()
	i.data[key] = &State{
		status:  res.status,
		Status:  res.Status,
		Count:   res.Count,
		Total:   res.Total,
		CloseAt: res.CloseAt,
		ResetAt: res.ResetAt,
	}
	i.rw.Unlock()

	return nil
}
