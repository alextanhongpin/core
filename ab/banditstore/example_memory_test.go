package banditstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/ab"
	"github.com/alextanhongpin/core/ab/banditstore"
)

func TestMemoryStoreExample(t *testing.T) {
	ctx := context.Background()
	store := banditstore.NewMemoryStore()

	// Create a new experiment
	exp := &ab.BanditExperiment{
		ID:        "exp1",
		Name:      "Homepage Button Test",
		Algorithm: ab.EpsilonGreedy,
		Arms: []ab.BanditArm{
			{ID: "A", Name: "Blue Button"},
			{ID: "B", Name: "Red Button"},
		},
	}
	if err := store.CreateExperiment(ctx, exp); err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// Retrieve the experiment
	got, err := store.GetExperiment(ctx, "exp1")
	if err != nil {
		t.Fatalf("GetExperiment failed: %v", err)
	}
	fmt.Printf("Experiment: %s, Arms: %v\n", got.Name, got.Arms)

	// Add a result
	result := ab.BanditResult{
		ExperimentID: "exp1",
		ArmID:        "A",
		UserID:       "user-123",
		Reward:       1.0,
		Success:      true,
		Timestamp:    time.Now(),
	}
	if err := store.AddResult(ctx, result); err != nil {
		t.Fatalf("AddResult failed: %v", err)
	}

	// List results
	results, err := store.ListResults(ctx, "exp1")
	if err != nil {
		t.Fatalf("ListResults failed: %v", err)
	}
	fmt.Printf("Results: %+v\n", results)
}
