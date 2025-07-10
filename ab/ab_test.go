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

func TestRecommendationEngine(t *testing.T) {
	ctx := context.Background()
	engine := NewRecommendationEngine()

	// Add users
	user1 := &User{
		ID: "user1",
		Preferences: map[string]float64{
			"action": 0.8,
			"comedy": 0.3,
			"drama":  0.6,
		},
		Demographics: map[string]string{
			"age":    "25-34",
			"gender": "M",
		},
	}
	engine.AddUser(user1)

	// Add items
	item1 := &Item{
		ID:       "movie1",
		Title:    "Action Movie",
		Category: "movie",
		Features: map[string]float64{
			"action": 0.9,
			"comedy": 0.1,
			"drama":  0.2,
		},
		Popularity: 8.5,
		CreatedAt:  time.Now(),
	}
	engine.AddItem(item1)

	item2 := &Item{
		ID:       "movie2",
		Title:    "Comedy Movie",
		Category: "movie",
		Features: map[string]float64{
			"action": 0.1,
			"comedy": 0.9,
			"drama":  0.3,
		},
		Popularity: 7.2,
		CreatedAt:  time.Now(),
	}
	engine.AddItem(item2)

	// Record interactions
	interaction := Interaction{
		UserID:    "user1",
		ItemID:    "movie1",
		Type:      "view",
		Rating:    4.5,
		Timestamp: time.Now(),
	}
	engine.RecordInteraction(interaction)

	// Get content-based recommendations
	recommendations, err := engine.GetRecommendations(ctx, "user1", 5, RecommendationContentBased)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(recommendations), 1)

	// Verify recommendation structure
	rec := recommendations[0]
	assert.Equal(t, "user1", rec.UserID)
	assert.NotEmpty(t, rec.ItemID)
	assert.Greater(t, rec.Score, 0.0)
	assert.Equal(t, RecommendationContentBased, rec.Algorithm)
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

func TestAnalyticsEngine(t *testing.T) {
	ctx := context.Background()
	engine := NewAnalyticsEngine()

	// Create metrics
	conversionMetric := &Metric{
		ID:          "conversion_rate",
		Name:        "Conversion Rate",
		Type:        MetricConversion,
		Description: "Percentage of users who convert",
		Unit:        "percent",
		IsPrimary:   true,
	}
	err := engine.CreateMetric(conversionMetric)
	require.NoError(t, err)

	revenueMetric := &Metric{
		ID:          "revenue",
		Name:        "Revenue",
		Type:        MetricRevenue,
		Description: "Total revenue generated",
		Unit:        "dollars",
		IsPrimary:   false,
	}
	err = engine.CreateMetric(revenueMetric)
	require.NoError(t, err)

	// Add some assignments to simulate experiment participation
	engine.assignments["user1_test_exp_1"] = &Assignment{
		UserID:       "user1",
		ExperimentID: "test_exp_1",
		VariantID:    "control",
		AssignedAt:   time.Now(),
	}
	engine.assignments["user2_test_exp_1"] = &Assignment{
		UserID:       "user2",
		ExperimentID: "test_exp_1",
		VariantID:    "treatment",
		AssignedAt:   time.Now(),
	}

	// Record metric values
	timeRange := TimeRange{
		Start: time.Now().Add(-2 * time.Hour), // Wider time range
		End:   time.Now().Add(1 * time.Hour),  // Future end time to capture all
	}

	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("user%d", (i%2)+1)
		variantID := "control"
		if i%2 == 1 {
			variantID = "treatment"
		}

		// Conversion events
		conversionValue := MetricValue{
			MetricID:     "conversion_rate",
			ExperimentID: "test_exp_1",
			VariantID:    variantID,
			UserID:       userID,
			Value:        float64(i % 2), // 50% conversion rate
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Minute),
		}
		err = engine.RecordMetricValue(ctx, conversionValue)
		require.NoError(t, err)

		// Revenue events
		if i%2 == 0 { // Only for conversions
			revenueValue := MetricValue{
				MetricID:     "revenue",
				ExperimentID: "test_exp_1",
				VariantID:    variantID,
				UserID:       userID,
				Value:        10.0 + float64(i%50), // Revenue between $10-60
				Timestamp:    time.Now().Add(-time.Duration(i) * time.Minute),
			}
			err = engine.RecordMetricValue(ctx, revenueValue)
			require.NoError(t, err)
		}
	}

	// Generate analytics
	analytics, err := engine.GenerateAnalytics(ctx, "test_exp_1", timeRange)
	require.NoError(t, err)
	assert.Equal(t, "test_exp_1", analytics.ExperimentID)

	// Debug output
	t.Logf("Overall metrics count: %d", len(analytics.OverallMetrics))
	t.Logf("Time series data count: %d", len(analytics.TimeSeriesData))
	for i, metric := range analytics.OverallMetrics {
		t.Logf("Metric %d: %s, Count: %d", i, metric.MetricName, metric.Count)
	}

	assert.GreaterOrEqual(t, len(analytics.OverallMetrics), 1)
	assert.GreaterOrEqual(t, len(analytics.TimeSeriesData), 1)
}

func TestConfigManager(t *testing.T) {
	ctx := context.Background()
	provider := NewInMemoryConfigProvider()
	manager := NewConfigManager(provider)

	// Create feature flag
	flag := &FeatureFlag{
		ID:          "new_checkout",
		Name:        "New Checkout Flow",
		Description: "Enable new checkout experience",
		Enabled:     true,
		Rules: []FeatureFlagRule{
			{
				ID:   "premium_users",
				Name: "Premium Users Only",
				Conditions: []RuleCondition{
					{
						Attribute: "subscription",
						Operator:  "equals",
						Value:     "premium",
					},
				},
				Action: RuleAction{
					Type:  "enable",
					Value: true,
				},
				Priority: 100,
			},
		},
		Rollout: &RolloutConfig{
			Strategy:   "percentage",
			Percentage: 50.0,
		},
	}

	err := manager.CreateFeatureFlag(ctx, flag)
	require.NoError(t, err)

	// Evaluate feature flag
	userAttributes := map[string]interface{}{
		"subscription": "premium",
		"country":      "US",
	}

	result, err := manager.EvaluateFeatureFlag(ctx, "new_checkout", "user123", userAttributes)
	require.NoError(t, err)
	assert.Equal(t, "new_checkout", result.FlagID)
	assert.Equal(t, "user123", result.UserID)

	// Create experiment config
	expConfig := &ExperimentConfig{
		ExperimentID: "checkout_test",
		TrafficSplit: map[string]float64{
			"control":   50.0,
			"treatment": 50.0,
		},
		TargetingRules: []TargetingRule{
			{
				ID:   "us_users",
				Name: "US Users",
				Conditions: []RuleCondition{
					{
						Attribute: "country",
						Operator:  "equals",
						Value:     "US",
					},
				},
				Include:  true,
				Priority: 1,
			},
		},
		StoppingRules: []StoppingRule{
			{
				ID:   "significance",
				Type: "significance",
				Condition: map[string]interface{}{
					"p_value":   0.05,
					"min_users": 1000,
				},
				Action: "stop",
			},
		},
	}

	err = manager.CreateExperimentConfig(ctx, expConfig)
	require.NoError(t, err)

	// Retrieve config
	retrievedConfig, err := manager.GetExperimentConfig(ctx, "checkout_test")
	require.NoError(t, err)
	assert.Equal(t, "checkout_test", retrievedConfig.ExperimentID)
	assert.Len(t, retrievedConfig.TrafficSplit, 2)
}

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup all engines
	store := NewInMemoryExperimentStore()
	experimentEngine := NewExperimentEngine(store)
	recommendationEngine := NewRecommendationEngine()
	analyticsEngine := NewAnalyticsEngine()
	configProvider := NewInMemoryConfigProvider()
	configManager := NewConfigManager(configProvider)

	// Create a comprehensive test scenario

	// 1. Setup experiment
	exp := &Experiment{
		ID:          "integration_test",
		Name:        "Homepage Recommendations",
		Description: "Testing different recommendation algorithms",
		Status:      StatusActive,
		Variants: []Variant{
			{ID: "control", Name: "Popularity Based", Weight: 0.33},
			{ID: "collaborative", Name: "Collaborative Filtering", Weight: 0.33},
			{ID: "content", Name: "Content Based", Weight: 0.34},
		},
		TargetPercentage: 1.0,
	}

	err := experimentEngine.CreateExperiment(exp)
	require.NoError(t, err)

	// 2. Setup feature flag
	flag := &FeatureFlag{
		ID:      "ml_recommendations",
		Name:    "ML Recommendations",
		Enabled: true,
		Rollout: &RolloutConfig{
			Strategy:   "percentage",
			Percentage: 100.0,
		},
	}

	err = configManager.CreateFeatureFlag(ctx, flag)
	require.NoError(t, err)

	// 3. Setup users and items for recommendations
	users := []*User{
		{ID: "user1", Preferences: map[string]float64{"tech": 0.8, "sports": 0.2}},
		{ID: "user2", Preferences: map[string]float64{"tech": 0.3, "sports": 0.7}},
		{ID: "user3", Preferences: map[string]float64{"tech": 0.6, "sports": 0.4}},
	}

	for _, user := range users {
		recommendationEngine.AddUser(user)
	}

	items := []*Item{
		{ID: "item1", Title: "Tech Article", Features: map[string]float64{"tech": 0.9, "sports": 0.1}},
		{ID: "item2", Title: "Sports News", Features: map[string]float64{"tech": 0.1, "sports": 0.9}},
		{ID: "item3", Title: "Tech Review", Features: map[string]float64{"tech": 0.8, "sports": 0.2}},
	}

	for _, item := range items {
		recommendationEngine.AddItem(item)
	}

	// 4. Setup metrics
	ctrMetric := &Metric{
		ID:        "ctr",
		Name:      "Click Through Rate",
		Type:      MetricClickThrough,
		IsPrimary: true,
	}
	err = analyticsEngine.CreateMetric(ctrMetric)
	require.NoError(t, err)

	// 5. Run simulation
	for i := 0; i < 30; i++ {
		userID := fmt.Sprintf("user%d", (i%3)+1)

		// Check feature flag
		flagResult, err := configManager.EvaluateFeatureFlag(ctx, "ml_recommendations", userID, nil)
		require.NoError(t, err)

		if flagResult.Enabled {
			// Assign to experiment variant
			assignment, err := experimentEngine.AssignVariant(ctx, userID, "integration_test", nil)
			require.NoError(t, err)

			if assignment != nil {
				// Get recommendations based on variant
				var recType RecommendationType
				switch assignment.VariantID {
				case "control":
					recType = RecommendationPopularity
				case "collaborative":
					recType = RecommendationCollaborative
				case "content":
					recType = RecommendationContentBased
				}

				recommendations, err := recommendationEngine.GetRecommendations(ctx, userID, 3, recType)
				require.NoError(t, err)

				// Simulate user interaction (click)
				if len(recommendations) > 0 {
					clicked := i%3 == 0 // 33% click rate

					// Record metric
					metricValue := MetricValue{
						MetricID:     "ctr",
						ExperimentID: "integration_test",
						VariantID:    assignment.VariantID,
						UserID:       userID,
						Value:        map[bool]float64{true: 1.0, false: 0.0}[clicked],
						Timestamp:    time.Now(),
					}

					err = analyticsEngine.RecordMetricValue(ctx, metricValue)
					require.NoError(t, err)

					// Record conversion if clicked
					if clicked {
						conversionEvent := &ConversionEvent{
							UserID:       userID,
							ExperimentID: "integration_test",
							VariantID:    assignment.VariantID,
							EventType:    "click",
							Value:        1.0,
							Timestamp:    time.Now(),
						}

						err = experimentEngine.RecordConversion(ctx, conversionEvent)
						require.NoError(t, err)
					}
				}
			}
		}
	}

	// 6. Analyze results
	results, err := experimentEngine.GetExperimentResults("integration_test")
	require.NoError(t, err)
	assert.Equal(t, "integration_test", results.ExperimentID)
	assert.Len(t, results.Variants, 3)

	// Generate analytics
	timeRange := TimeRange{
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	}

	analytics, err := analyticsEngine.GenerateAnalytics(ctx, "integration_test", timeRange)
	require.NoError(t, err)
	assert.Equal(t, "integration_test", analytics.ExperimentID)

	// Verify we have data
	totalImpressions := int64(0)
	for _, variant := range results.Variants {
		totalImpressions += variant.Impressions
	}
	assert.Greater(t, totalImpressions, int64(0))

	t.Logf("Integration test completed successfully!")
	t.Logf("Total impressions: %d", totalImpressions)
	t.Logf("Variants tested: %d", len(results.Variants))
	t.Logf("Analytics data points: %d", len(analytics.TimeSeriesData))
}
