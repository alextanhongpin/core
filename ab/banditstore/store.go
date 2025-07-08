package banditstore

import (
	"context"

	"github.com/alextanhongpin/core/ab"
)

// ExperimentStore defines persistence for bandit experiments and arms.
type ExperimentStore interface {
	CreateExperiment(ctx context.Context, exp *ab.BanditExperiment) error
	UpdateExperiment(ctx context.Context, exp *ab.BanditExperiment) error
	GetExperiment(ctx context.Context, id string) (*ab.BanditExperiment, error)
	ListExperiments(ctx context.Context) ([]*ab.BanditExperiment, error)
}

// ResultStore defines persistence for bandit results.
type ResultStore interface {
	AddResult(ctx context.Context, result ab.BanditResult) error
	ListResults(ctx context.Context, experimentID string) ([]ab.BanditResult, error)
}

// Store combines experiment and result storage.
type Store interface {
	ExperimentStore
	ResultStore
}
