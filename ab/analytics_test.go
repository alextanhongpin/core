package ab_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ab "github.com/alextanhongpin/core/ab"
)

func TestAnalyticsStorage_InMemory(t *testing.T) {
	ctx := context.Background()
	storage := ab.NewInMemoryAnalyticsStorage()
	mv := ab.MetricValue{
		MetricID:     "conversion",
		ExperimentID: "exp1",
		VariantID:    "A",
		UserID:       "user1",
		Value:        0.5,
		Timestamp:    time.Now(),
	}
	assert.NoError(t, storage.SaveMetricValue(ctx, mv))
	timeRange := ab.TimeRange{Start: mv.Timestamp.Add(-time.Minute), End: mv.Timestamp.Add(time.Minute)}
	list, err := storage.GetMetricValues(ctx, "exp1", timeRange)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, mv.Value, list[0].Value)
}

func TestAnalyticsEngine_InMemory(t *testing.T) {
	engine := ab.NewAnalyticsEngine()
	ctx := context.Background()

	metric := &ab.Metric{
		ID:          "conversion_rate",
		Name:        "Conversion Rate",
		Type:        ab.MetricConversion,
		Description: "Test conversion rate",
		Unit:        "percent",
		IsPrimary:   true,
	}
	assert.NoError(t, engine.CreateMetric(metric))

	// Record metric values
	now := time.Now()
	for i := 0; i < 10; i++ {
		userID := "user" + string(rune('A'+i))
		variant := "A"
		if i%2 == 1 {
			variant = "B"
		}
		val := float64(i % 2)
		mv := ab.MetricValue{
			MetricID:     "conversion_rate",
			ExperimentID: "exp1",
			VariantID:    variant,
			UserID:       userID,
			Value:        val,
			Timestamp:    now.Add(time.Duration(i) * time.Minute),
		}
		assert.NoError(t, engine.RecordMetricValue(ctx, mv))
	}

	timeRange := ab.TimeRange{Start: now.Add(-time.Hour), End: now.Add(time.Hour)}
	analytics, err := engine.GenerateAnalytics(ctx, "exp1", timeRange)
	assert.NoError(t, err)
	assert.Equal(t, "exp1", analytics.ExperimentID)
	assert.GreaterOrEqual(t, len(analytics.OverallMetrics), 1)
}
