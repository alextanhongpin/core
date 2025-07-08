package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/ab"
)

func ExampleExperimentStoreUsage() {
	store := ab.NewInMemoryExperimentStore()
	engine := ab.NewExperimentEngine(store)

	// Create an experiment
	exp := &ab.Experiment{
		Name:   "Homepage Button Test",
		Status: ab.StatusActive,
		Variants: []ab.Variant{
			{ID: "A", Name: "Blue Button", Weight: 0.5},
			{ID: "B", Name: "Red Button", Weight: 0.5},
		},
		TargetPercentage: 1.0,
		ConfidenceLevel:  0.95,
	}
	if err := engine.CreateExperiment(exp); err != nil {
		panic(err)
	}

	// Assign a user
	assignment, err := engine.AssignVariant(context.Background(), "user123", exp.ID, map[string]string{"country": "US"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("User assigned: %+v\n", assignment)

	// Record a conversion
	conv := &ab.ConversionEvent{
		UserID:       "user123",
		ExperimentID: exp.ID,
		VariantID:    assignment.VariantID,
		EventType:    "purchase",
		Value:        99.99,
		Timestamp:    time.Now(),
	}
	if err := engine.RecordConversion(context.Background(), conv); err != nil {
		panic(err)
	}

	// List assignments
	assignments, _ := store.ListAssignments(exp.ID)
	fmt.Printf("Assignments: %d\n", len(assignments))

	// List conversions
	conversions, _ := store.ListConversions(exp.ID)
	fmt.Printf("Conversions: %d\n", len(conversions))

	// List experiments
	experiments, _ := store.ListExperiments()
	fmt.Printf("Experiments: %d\n", len(experiments))
}
