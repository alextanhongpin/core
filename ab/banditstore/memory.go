package banditstore

import (
	"context"
	"sync"

	"github.com/alextanhongpin/core/ab"
)

type memoryStore struct {
	experiments map[string]*ab.BanditExperiment
	results     map[string][]ab.BanditResult
	mu          sync.RWMutex
}

func NewMemoryStore() Store {
	return &memoryStore{
		experiments: make(map[string]*ab.BanditExperiment),
		results:     make(map[string][]ab.BanditResult),
	}
}

func (m *memoryStore) CreateExperiment(ctx context.Context, exp *ab.BanditExperiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.experiments[exp.ID] = exp
	return nil
}

func (m *memoryStore) UpdateExperiment(ctx context.Context, exp *ab.BanditExperiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.experiments[exp.ID] = exp
	return nil
}

func (m *memoryStore) GetExperiment(ctx context.Context, id string) (*ab.BanditExperiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, ok := m.experiments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return exp, nil
}

func (m *memoryStore) ListExperiments(ctx context.Context) ([]*ab.BanditExperiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exps := make([]*ab.BanditExperiment, 0, len(m.experiments))
	for _, exp := range m.experiments {
		exps = append(exps, exp)
	}
	return exps, nil
}

func (m *memoryStore) AddResult(ctx context.Context, result ab.BanditResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[result.ExperimentID] = append(m.results[result.ExperimentID], result)
	return nil
}

func (m *memoryStore) ListResults(ctx context.Context, experimentID string) ([]ab.BanditResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ab.BanditResult(nil), m.results[experimentID]...), nil
}
