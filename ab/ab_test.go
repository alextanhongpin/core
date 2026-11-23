package ab

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicABTesting(t *testing.T) {
	// Test basic hash and rollout functionality
	userID := "user123"

	// Test consistent hashing
	hash1 := Hash(userID, 100)
	hash2 := Hash(userID, 100)
	assert.Equal(t, hash1, hash2, "Hash should be consistent")

	// Test rollout
	included := Rollout(userID, 50) // 50% rollout
	assert.IsType(t, true, included, "Rollout should return boolean")
}

func TestExperimentEngine(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryExperimentStore()
	engine := NewExperimentEngine(store)

	// Create an experiment
	exp := &Experiment{
		ID:          "test_exp_1",
		Name:        "Button Color Test",
		Description: "Testing different button colors",
		Status:      StatusActive,
		Variants: []Variant{
			{ID: "control", Name: "Blue Button", Weight: 0.5},
			{ID: "treatment", Name: "Red Button", Weight: 0.5},
		},
		TargetPercentage:    1.0, // 100% of users
		MinSampleSize:       100,
		ConfidenceLevel:     0.95,
		MinDetectableEffect: 0.05,
	}

	err := engine.CreateExperiment(exp)
	require.NoError(t, err)

	// Assign users to variants
	userAttributes := map[string]string{"country": "US", "device": "mobile"}

	assignment1, err := engine.AssignVariant(ctx, "user1", "test_exp_1", userAttributes)
	require.NoError(t, err)
	assert.NotNil(t, assignment1)
	assert.Contains(t, []string{"control", "treatment"}, assignment1.VariantID)

	// Same user should get same assignment
	assignment2, err := engine.AssignVariant(ctx, "user1", "test_exp_1", userAttributes)
	require.NoError(t, err)
	assert.Equal(t, assignment1.VariantID, assignment2.VariantID)

	// Record conversion
	conversionEvent := &ConversionEvent{
		UserID:       "user1",
		ExperimentID: "test_exp_1",
		VariantID:    assignment1.VariantID,
		EventType:    "button_click",
		Value:        1.0,
		Timestamp:    time.Now(),
	}

	err = engine.RecordConversion(ctx, conversionEvent)
	require.NoError(t, err)

	// Get results
	results, err := engine.GetExperimentResults("test_exp_1")
	require.NoError(t, err)
	assert.Equal(t, "test_exp_1", results.ExperimentID)
	assert.Len(t, results.Variants, 2)
}


func TestBanditEngine(t *testing.T) {
	engine := NewBanditEngine()

	// Create bandit experiment
	exp := &BanditExperiment{
		ID:        "bandit_test_1",
		Name:      "Ad Banner Test",
		Algorithm: EpsilonGreedy,
		Arms: []BanditArm{
			{ID: "banner_a", Name: "Banner A", Alpha: 1, Beta: 1},
			{ID: "banner_b", Name: "Banner B", Alpha: 1, Beta: 1},
			{ID: "banner_c", Name: "Banner C", Alpha: 1, Beta: 1},
		},
		Epsilon: 0.1,
	}

	err := engine.CreateBanditExperiment(exp)
	require.NoError(t, err)

	// Select arms and record rewards
	for i := 0; i < 50; i++ {
		userID := fmt.Sprintf("user%d", i)

		// Select arm
		arm, err := engine.SelectArm("bandit_test_1", userID, nil)
		require.NoError(t, err)
		assert.NotNil(t, arm)

		// Simulate reward (banner A performs better)
		reward := 0.0
		success := false
		if arm.ID == "banner_a" {
			success = i%3 == 0 // 33% success rate
		} else {
			success = i%5 == 0 // 20% success rate
		}

		if success {
			reward = 1.0
		}

		// Record reward
		err = engine.RecordReward("bandit_test_1", arm.ID, userID, reward, success, nil)
		require.NoError(t, err)
	}

	// Get analysis
	analysis, err := engine.GetBanditResults("bandit_test_1")
	require.NoError(t, err)
	assert.Equal(t, "bandit_test_1", analysis.ExperimentID)
	assert.Len(t, analysis.Arms, 3)

	// Get recommendation
	recommendation, err := engine.GetExperimentRecommendation("bandit_test_1")
	require.NoError(t, err)
	assert.NotEmpty(t, recommendation.Decision)
}
