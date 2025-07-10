package ab_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ab "github.com/alextanhongpin/core/ab"
)

func TestConfigStorage_InMemory(t *testing.T) {
	storage := ab.NewInMemoryConfigStorage()
	cfg := ab.ExperimentConfig{
		ExperimentID: "exp1",
		TrafficSplit: map[string]float64{"A": 50, "B": 50},
	}
	assert.NoError(t, storage.SaveExperimentConfig(&cfg))
	list, err := storage.ListExperimentConfigs()
	assert.NoError(t, err)
	var found *ab.ExperimentConfig
	for _, c := range list {
		if c.ExperimentID == "exp1" {
			found = c
		}
	}
	assert.NotNil(t, found)
	assert.Equal(t, 2, len(found.TrafficSplit))
}
