package ab_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ab "github.com/alextanhongpin/core/ab"
)

func TestBanditEngine_BasicFlow(t *testing.T) {
	engine := ab.NewBanditEngine()
	exp := &ab.BanditExperiment{
		Name:      "TestExp",
		Algorithm: ab.EpsilonGreedy,
		Arms:      []ab.BanditArm{{ID: "A", Name: "A"}, {ID: "B", Name: "B"}},
	}
	err := engine.CreateBanditExperiment(exp)
	require.NoError(t, err)
	assert.NotEmpty(t, exp.ID)

	// Select arm
	arm, err := engine.SelectArm(exp.ID, "user1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, arm)

	// Record reward
	err = engine.RecordReward(exp.ID, arm.ID, "user1", 1.0, true, nil)
	assert.NoError(t, err)

	// Get results
	analysis, err := engine.GetBanditResults(exp.ID)
	assert.NoError(t, err)
	assert.Equal(t, exp.ID, analysis.ExperimentID)
	assert.Len(t, analysis.Arms, 2)
}

func TestBanditEngine_StopExperiment(t *testing.T) {
	engine := ab.NewBanditEngine()
	exp := &ab.BanditExperiment{
		Name:      "TestExp2",
		Algorithm: ab.UCB,
		Arms:      []ab.BanditArm{{ID: "A", Name: "A"}, {ID: "B", Name: "B"}},
	}
	err := engine.CreateBanditExperiment(exp)
	require.NoError(t, err)
	err = engine.StopExperiment(exp.ID)
	assert.NoError(t, err)
	assert.Equal(t, "stopped", exp.Status)
	assert.NotNil(t, exp.EndTime)
}

func TestBanditEngine_Recommendation(t *testing.T) {
	engine := ab.NewBanditEngine()
	exp := &ab.BanditExperiment{
		Name:      "TestExp3",
		Algorithm: ab.ThompsonSampling,
		Arms:      []ab.BanditArm{{ID: "A", Name: "A"}, {ID: "B", Name: "B"}},
	}
	err := engine.CreateBanditExperiment(exp)
	require.NoError(t, err)
	// Not enough data
	rec, err := engine.GetExperimentRecommendation(exp.ID)
	assert.NoError(t, err)
	assert.Equal(t, "continue", rec.Decision)
	// Simulate pulls
	for i := 0; i < 120; i++ {
		engine.RecordReward(exp.ID, "A", "user", 1, true, nil)
		engine.RecordReward(exp.ID, "B", "user", 0, false, nil)
	}
	rec, err = engine.GetExperimentRecommendation(exp.ID)
	assert.NoError(t, err)
	assert.True(t, rec.Decision == "stop" || rec.Decision == "continue")
}
