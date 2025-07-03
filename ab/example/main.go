package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/ab"
)

func main() {
	ctx := context.Background()

	// Initialize all engines
	experimentEngine := ab.NewExperimentEngine()
	recommendationEngine := ab.NewRecommendationEngine()
	banditEngine := ab.NewBanditEngine()
	analyticsEngine := ab.NewAnalyticsEngine()
	configProvider := ab.NewInMemoryConfigProvider()
	configManager := ab.NewConfigManager(configProvider)

	fmt.Println("🚀 Starting Enhanced A/B Testing Framework Demo")

	// Demo 1: Feature Flags
	fmt.Println("\n📝 Demo 1: Feature Flags")
	demoFeatureFlags(ctx, configManager)

	// Demo 2: A/B Testing
	fmt.Println("\n🧪 Demo 2: A/B Testing")
	demoABTesting(ctx, experimentEngine, analyticsEngine)

	// Demo 3: Multi-Armed Bandits
	fmt.Println("\n🎰 Demo 3: Multi-Armed Bandits")
	demoBandits(banditEngine)

	// Demo 4: Recommendation System
	fmt.Println("\n🎯 Demo 4: Recommendation System")
	demoRecommendations(ctx, recommendationEngine)

	// Demo 5: Integrated Analytics
	fmt.Println("\n📊 Demo 5: Integrated Analytics")
	demoAnalytics(ctx, analyticsEngine)

	fmt.Println("\n✅ Demo completed successfully!")
}

func demoFeatureFlags(ctx context.Context, manager *ab.ConfigManager) {
	// Create a feature flag for premium features
	flag := &ab.FeatureFlag{
		ID:          "premium_dashboard",
		Name:        "Premium Dashboard",
		Description: "Enhanced dashboard for premium users",
		Enabled:     true,
		Rules: []ab.FeatureFlagRule{
			{
				ID:   "premium_users",
				Name: "Premium Users Only",
				Conditions: []ab.RuleCondition{
					{
						Attribute: "subscription",
						Operator:  "equals",
						Value:     "premium",
					},
				},
				Action: ab.RuleAction{
					Type:  "enable",
					Value: true,
					Config: map[string]interface{}{
						"theme":    "premium",
						"features": []string{"advanced_analytics", "priority_support"},
					},
				},
				Priority: 100,
			},
		},
		Rollout: &ab.RolloutConfig{
			Strategy:   "percentage",
			Percentage: 80.0, // 80% rollout
		},
	}

	err := manager.CreateFeatureFlag(ctx, flag)
	if err != nil {
		log.Printf("Error creating feature flag: %v", err)
		return
	}

	// Test different user types
	users := []struct {
		id         string
		attributes map[string]interface{}
	}{
		{"user1", map[string]interface{}{"subscription": "premium", "country": "US"}},
		{"user2", map[string]interface{}{"subscription": "basic", "country": "US"}},
		{"user3", map[string]interface{}{"subscription": "premium", "country": "UK"}},
	}

	for _, user := range users {
		result, err := manager.EvaluateFeatureFlag(ctx, "premium_dashboard", user.id, user.attributes)
		if err != nil {
			log.Printf("Error evaluating flag for %s: %v", user.id, err)
			continue
		}

		fmt.Printf("  User %s (subscription: %s): Enabled=%v, Reason=%s\n",
			user.id, user.attributes["subscription"], result.Enabled, result.Reason)

		if result.Enabled && len(result.Config) > 0 {
			fmt.Printf("    Config: %+v\n", result.Config)
		}
	}
}

func demoABTesting(ctx context.Context, engine *ab.ExperimentEngine, analytics *ab.AnalyticsEngine) {
	// Create an A/B test for button colors
	exp := &ab.Experiment{
		ID:          "button_color_test",
		Name:        "Homepage Button Color Test",
		Description: "Testing the impact of button color on conversion rates",
		Status:      ab.StatusActive,
		Variants: []ab.Variant{
			{ID: "control", Name: "Blue Button", Weight: 0.4},
			{ID: "treatment_a", Name: "Red Button", Weight: 0.3},
			{ID: "treatment_b", Name: "Green Button", Weight: 0.3},
		},
		TargetPercentage:    1.0,
		MinSampleSize:       50,
		ConfidenceLevel:     0.95,
		MinDetectableEffect: 0.05,
	}

	err := engine.CreateExperiment(exp)
	if err != nil {
		log.Printf("Error creating experiment: %v", err)
		return
	}

	// Setup metrics
	conversionMetric := &ab.Metric{
		ID:          "button_click",
		Name:        "Button Click Rate",
		Type:        ab.MetricConversion,
		Description: "Percentage of users who click the button",
		Unit:        "percent",
		IsPrimary:   true,
	}
	analytics.CreateMetric(conversionMetric)

	// Simulate users visiting the page
	fmt.Printf("  Simulating 200 user visits...\n")

	for i := 0; i < 200; i++ {
		userID := fmt.Sprintf("user_%d", i)
		userAttributes := map[string]string{
			"country": []string{"US", "UK", "CA", "AU"}[i%4],
			"device":  []string{"mobile", "desktop", "tablet"}[i%3],
		}

		// Assign user to variant
		assignment, err := engine.AssignVariant(ctx, userID, "button_color_test", userAttributes)
		if err != nil || assignment == nil {
			continue
		}

		// Simulate conversion based on variant (treatment_b performs better)
		var conversionRate float64
		switch assignment.VariantID {
		case "control":
			conversionRate = 0.15 // 15% base rate
		case "treatment_a":
			conversionRate = 0.12 // 12% (red performs worse)
		case "treatment_b":
			conversionRate = 0.22 // 22% (green performs better)
		}

		// Add some randomness
		converted := rand.Float64() < conversionRate

		// Record metric
		metricValue := ab.MetricValue{
			MetricID:     "button_click",
			ExperimentID: "button_color_test",
			VariantID:    assignment.VariantID,
			UserID:       userID,
			Value:        map[bool]float64{true: 1.0, false: 0.0}[converted],
			Timestamp:    time.Now(),
		}
		analytics.RecordMetricValue(ctx, metricValue)

		// Record conversion in experiment engine
		if converted {
			conversionEvent := &ab.ConversionEvent{
				UserID:       userID,
				ExperimentID: "button_color_test",
				VariantID:    assignment.VariantID,
				EventType:    "button_click",
				Value:        1.0,
				Timestamp:    time.Now(),
			}
			engine.RecordConversion(ctx, conversionEvent)
		}
	}

	// Get experiment results
	results, err := engine.GetExperimentResults("button_color_test")
	if err != nil {
		log.Printf("Error getting results: %v", err)
		return
	}

	fmt.Printf("  Experiment Results:\n")
	for _, variant := range results.Variants {
		fmt.Printf("    %s: %.1f%% conversion rate (%d/%d users)",
			variant.VariantID,
			variant.ConversionRate*100,
			variant.Conversions,
			variant.Impressions)

		if variant.IsSignificant {
			fmt.Printf(" ✅ SIGNIFICANT")
		}
		fmt.Println()
	}
}

func demoBandits(engine *ab.BanditEngine) {
	// Create a bandit experiment for ad optimization
	exp := &ab.BanditExperiment{
		ID:        "ad_banner_optimization",
		Name:      "Advertisement Banner Optimization",
		Algorithm: ab.ThompsonSampling,
		Arms: []ab.BanditArm{
			{ID: "banner_sports", Name: "Sports Banner", Alpha: 1, Beta: 1},
			{ID: "banner_tech", Name: "Tech Banner", Alpha: 1, Beta: 1},
			{ID: "banner_lifestyle", Name: "Lifestyle Banner", Alpha: 1, Beta: 1},
		},
	}

	err := engine.CreateBanditExperiment(exp)
	if err != nil {
		log.Printf("Error creating bandit experiment: %v", err)
		return
	}

	fmt.Printf("  Running bandit optimization for 100 iterations...\n")

	// Simulate different performance for each banner
	trueRates := map[string]float64{
		"banner_sports":    0.25, // 25% click rate
		"banner_tech":      0.18, // 18% click rate
		"banner_lifestyle": 0.12, // 12% click rate
	}

	armCounts := make(map[string]int)

	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("visitor_%d", i)

		// Select arm using bandit algorithm
		arm, err := engine.SelectArm("ad_banner_optimization", userID, nil)
		if err != nil {
			continue
		}

		armCounts[arm.ID]++

		// Simulate click based on true performance
		clicked := rand.Float64() < trueRates[arm.ID]
		reward := 0.0
		if clicked {
			reward = 1.0
		}

		// Record reward
		engine.RecordReward("ad_banner_optimization", arm.ID, userID, reward, clicked, nil)
	}

	// Get results
	analysis, err := engine.GetBanditResults("ad_banner_optimization")
	if err != nil {
		log.Printf("Error getting bandit results: %v", err)
		return
	}

	fmt.Printf("  Bandit Results:\n")
	for _, arm := range analysis.Arms {
		fmt.Printf("    %s: %.1f%% conversion rate (%d pulls, %d conversions)\n",
			arm.ArmID,
			arm.ConversionRate*100,
			arm.Pulls,
			arm.Rewards)
	}

	fmt.Printf("  Arm Selection Distribution:\n")
	for armID, count := range armCounts {
		fmt.Printf("    %s: %d%% of traffic\n", armID, count)
	}
}

func demoRecommendations(ctx context.Context, engine *ab.RecommendationEngine) {
	// Add sample users with different preferences
	users := []*ab.User{
		{
			ID: "alice",
			Preferences: map[string]float64{
				"action": 0.9, "comedy": 0.2, "drama": 0.6, "sci-fi": 0.8,
			},
			Demographics: map[string]string{"age": "25-34", "gender": "F"},
		},
		{
			ID: "bob",
			Preferences: map[string]float64{
				"action": 0.3, "comedy": 0.8, "drama": 0.4, "sci-fi": 0.2,
			},
			Demographics: map[string]string{"age": "35-44", "gender": "M"},
		},
	}

	for _, user := range users {
		engine.AddUser(user)
	}

	// Add sample movies
	movies := []*ab.Item{
		{
			ID: "movie1", Title: "Space Warriors",
			Features:   map[string]float64{"action": 0.9, "comedy": 0.1, "drama": 0.3, "sci-fi": 0.9},
			Popularity: 8.5,
		},
		{
			ID: "movie2", Title: "Comedy Central",
			Features:   map[string]float64{"action": 0.1, "comedy": 0.9, "drama": 0.2, "sci-fi": 0.1},
			Popularity: 7.2,
		},
		{
			ID: "movie3", Title: "Drama Queen",
			Features:   map[string]float64{"action": 0.2, "comedy": 0.3, "drama": 0.9, "sci-fi": 0.1},
			Popularity: 7.8,
		},
		{
			ID: "movie4", Title: "Action Hero",
			Features:   map[string]float64{"action": 0.9, "comedy": 0.2, "drama": 0.4, "sci-fi": 0.3},
			Popularity: 8.1,
		},
	}

	for _, movie := range movies {
		engine.AddItem(movie)
	}

	// Record some interactions
	interactions := []ab.Interaction{
		{UserID: "alice", ItemID: "movie1", Type: "view", Rating: 4.5},
		{UserID: "alice", ItemID: "movie4", Type: "like", Rating: 4.0},
		{UserID: "bob", ItemID: "movie2", Type: "view", Rating: 4.8},
		{UserID: "bob", ItemID: "movie3", Type: "view", Rating: 3.5},
	}

	for _, interaction := range interactions {
		engine.RecordInteraction(interaction)
	}

	// Generate recommendations for each user and algorithm
	algorithms := []ab.RecommendationType{
		ab.RecommendationContentBased,
		ab.RecommendationCollaborative,
		ab.RecommendationPopularity,
		ab.RecommendationHybrid,
	}

	for _, userID := range []string{"alice", "bob"} {
		fmt.Printf("  Recommendations for %s:\n", userID)

		for _, algorithm := range algorithms {
			recommendations, err := engine.GetRecommendations(ctx, userID, 3, algorithm)
			if err != nil {
				continue
			}

			fmt.Printf("    %s:\n", algorithm)
			for i, rec := range recommendations {
				if i >= 2 { // Show top 2
					break
				}

				// Find movie title
				movieTitle := rec.ItemID
				for _, movie := range movies {
					if movie.ID == rec.ItemID {
						movieTitle = movie.Title
						break
					}
				}

				fmt.Printf("      %s (Score: %.2f) - %s\n",
					movieTitle, rec.Score, rec.Reason)
			}
		}
		fmt.Println()
	}
}

func demoAnalytics(ctx context.Context, analytics *ab.AnalyticsEngine) {
	// Create multiple metrics
	metrics := []*ab.Metric{
		{
			ID: "page_views", Name: "Page Views", Type: ab.MetricEngagement,
			Description: "Number of page views", Unit: "count", IsPrimary: false,
		},
		{
			ID: "revenue", Name: "Revenue", Type: ab.MetricRevenue,
			Description: "Revenue generated", Unit: "dollars", IsPrimary: true,
		},
		{
			ID: "retention", Name: "7-Day Retention", Type: ab.MetricRetention,
			Description: "Users retained after 7 days", Unit: "percent", IsPrimary: false,
		},
	}

	for _, metric := range metrics {
		analytics.CreateMetric(metric)
	}

	// Create a segment
	segment := &ab.Segment{
		ID:   "premium_users",
		Name: "Premium Users",
		Criteria: []ab.SegmentCriteria{
			{Attribute: "subscription", Operator: "equals", Value: "premium"},
		},
	}
	analytics.CreateSegment(segment)

	// Generate sample data
	fmt.Printf("  Generating analytics data...\n")

	variants := []string{"control", "treatment"}
	now := time.Now()

	for i := 0; i < 500; i++ {
		userID := fmt.Sprintf("analytics_user_%d", i)
		variant := variants[i%2]
		timestamp := now.Add(-time.Duration(i) * time.Minute)

		// Page views (treatment gets more views)
		pageViews := 3.0
		if variant == "treatment" {
			pageViews = 4.2
		}
		pageViews += rand.Float64()*2 - 1 // Add noise

		analytics.RecordMetricValue(ctx, ab.MetricValue{
			MetricID: "page_views", ExperimentID: "analytics_demo",
			VariantID: variant, UserID: userID, Value: pageViews, Timestamp: timestamp,
		})

		// Revenue (treatment generates more revenue)
		if i%10 == 0 { // 10% of users generate revenue
			revenue := 15.0
			if variant == "treatment" {
				revenue = 22.0
			}
			revenue += rand.Float64()*10 - 5 // Add noise

			analytics.RecordMetricValue(ctx, ab.MetricValue{
				MetricID: "revenue", ExperimentID: "analytics_demo",
				VariantID: variant, UserID: userID, Value: revenue, Timestamp: timestamp,
			})
		}

		// Retention (treatment has better retention)
		if i%20 == 0 { // Check retention for subset of users
			retained := 0.0
			if variant == "control" && rand.Float64() < 0.65 {
				retained = 1.0
			} else if variant == "treatment" && rand.Float64() < 0.78 {
				retained = 1.0
			}

			analytics.RecordMetricValue(ctx, ab.MetricValue{
				MetricID: "retention", ExperimentID: "analytics_demo",
				VariantID: variant, UserID: userID, Value: retained, Timestamp: timestamp,
			})
		}
	}

	// Generate comprehensive analytics
	timeRange := ab.TimeRange{
		Start: now.Add(-10 * time.Hour),
		End:   now,
	}

	analyticsResults, err := analytics.GenerateAnalytics(ctx, "analytics_demo", timeRange)
	if err != nil {
		log.Printf("Error generating analytics: %v", err)
		return
	}

	fmt.Printf("  Analytics Results:\n")
	fmt.Printf("    Time Range: %s to %s\n",
		timeRange.Start.Format("15:04"), timeRange.End.Format("15:04"))

	// Overall metrics
	fmt.Printf("    Overall Metrics:\n")
	for _, metric := range analyticsResults.OverallMetrics {
		fmt.Printf("      %s: Mean=%.2f, Count=%d, StdDev=%.2f\n",
			metric.MetricName, metric.Mean, metric.Count, metric.StandardDev)
	}

	// Variant comparison
	fmt.Printf("    Variant Comparison:\n")
	for variantID, metrics := range analyticsResults.VariantMetrics {
		fmt.Printf("      %s:\n", variantID)
		for _, metric := range metrics {
			fmt.Printf("        %s: %.2f (CI: %.2f-%.2f)\n",
				metric.MetricName, metric.Mean,
				metric.ConfidenceInterval.Lower, metric.ConfidenceInterval.Upper)
		}
	}

	// Statistical tests
	if len(analyticsResults.StatisticalTests) > 0 {
		fmt.Printf("    Statistical Tests:\n")
		for _, test := range analyticsResults.StatisticalTests {
			significance := "Not Significant"
			if test.IsSignificant {
				significance = "✅ SIGNIFICANT"
			}
			fmt.Printf("      %s vs %s (%s): p=%.4f %s\n",
				test.ControlVariant, test.TreatmentVariant, test.MetricID,
				test.PValue, significance)
		}
	}

	fmt.Printf("    Time Series Data Points: %d\n", len(analyticsResults.TimeSeriesData))
}
