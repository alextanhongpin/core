# Enhanced A/B Testing and Recommendation Framework

This Go package provides a comprehensive A/B testing and recommendation system with advanced features for experimentation, analytics, and configuration management.

## Features

### 🧪 A/B Testing Engine
- **Experiment Management**: Create, manage, and analyze A/B tests
- **Consistent User Assignment**: Hash-based user assignment ensuring consistency
- **Statistical Analysis**: Built-in statistical significance testing
- **Multi-variate Testing**: Support for experiments with multiple variants
- **Quality Control**: Sample ratio mismatch detection and bias analysis

### 🎯 Multi-Armed Bandits
- **Multiple Algorithms**: Epsilon-greedy, UCB, Thompson Sampling, Bayesian bandits
- **Real-time Optimization**: Dynamic allocation based on performance
- **Regret Analysis**: Track and minimize exploration costs
- **Automated Stopping**: Smart recommendations for when to stop experiments

### 🔍 Recommendation Engine
- **Multiple Algorithms**: Content-based, collaborative filtering, popularity, trending
- **Hybrid Recommendations**: Combine multiple algorithms with configurable weights
- **Real-time Learning**: Update preferences based on user interactions
- **Contextual Recommendations**: Support for contextual information

### 📊 Advanced Analytics
- **Comprehensive Metrics**: Conversion, revenue, engagement, retention metrics
- **Statistical Testing**: T-tests, confidence intervals, effect size calculations
- **Segmentation Analysis**: Analyze performance across user segments
- **Time Series Data**: Track metrics over time for trend analysis
- **Export Capabilities**: JSON and CSV export formats

### ⚙️ Configuration Management
- **Feature Flags**: Dynamic feature toggling with targeting rules
- **Experiment Configuration**: Centralized experiment settings
- **Real-time Updates**: Watch for configuration changes
- **Rollout Control**: Gradual feature rollouts with percentage-based targeting

## Quick Start

### Basic A/B Testing

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/ab"
)

func main() {
    ctx := context.Background()
    engine := ab.NewExperimentEngine()
    
    // Create an experiment
    exp := &ab.Experiment{
        ID:          "button_color_test",
        Name:        "Button Color Experiment",
        Description: "Testing blue vs red button colors",
        Status:      ab.StatusActive,
        Variants: []ab.Variant{
            {ID: "control", Name: "Blue Button", Weight: 0.5},
            {ID: "treatment", Name: "Red Button", Weight: 0.5},
        },
        TargetPercentage: 1.0, // 100% of users
        ConfidenceLevel:  0.95,
    }
    
    engine.CreateExperiment(exp)
    
    // Assign a user to a variant
    userAttributes := map[string]string{"country": "US"}
    assignment, _ := engine.AssignVariant(ctx, "user123", "button_color_test", userAttributes)
    
    fmt.Printf("User assigned to variant: %s\n", assignment.VariantID)
    
    // Record a conversion
    conversion := &ab.ConversionEvent{
        UserID:       "user123",
        ExperimentID: "button_color_test",
        VariantID:    assignment.VariantID,
        EventType:    "button_click",
        Value:        1.0,
    }
    
    engine.RecordConversion(ctx, conversion)
    
    // Get results
    results, _ := engine.GetExperimentResults("button_color_test")
    fmt.Printf("Experiment results: %+v\n", results)
}
```

### Multi-Armed Bandits

```go
// Create a bandit experiment
banditEngine := ab.NewBanditEngine()

exp := &ab.BanditExperiment{
    ID:        "ad_banner_test",
    Name:      "Ad Banner Optimization",
    Algorithm: ab.EpsilonGreedy,
    Arms: []ab.BanditArm{
        {ID: "banner_a", Name: "Banner A"},
        {ID: "banner_b", Name: "Banner B"},
        {ID: "banner_c", Name: "Banner C"},
    },
    Epsilon: 0.1, // 10% exploration
}

banditEngine.CreateBanditExperiment(exp)

// Select an arm for a user
arm, _ := banditEngine.SelectArm("ad_banner_test", "user123", nil)
fmt.Printf("Selected arm: %s\n", arm.ID)

// Record reward
banditEngine.RecordReward("ad_banner_test", arm.ID, "user123", 1.0, true, nil)
```

### Recommendations

```go
// Create recommendation engine
recEngine := ab.NewRecommendationEngine()

// Add users and items
user := &ab.User{
    ID: "user123",
    Preferences: map[string]float64{
        "action": 0.8,
        "comedy": 0.3,
    },
}
recEngine.AddUser(user)

item := &ab.Item{
    ID:    "movie1",
    Title: "Action Movie",
    Features: map[string]float64{
        "action": 0.9,
        "comedy": 0.1,
    },
}
recEngine.AddItem(item)

// Get recommendations
recommendations, _ := recEngine.GetRecommendations(
    ctx, "user123", 5, ab.RecommendationContentBased)

for _, rec := range recommendations {
    fmt.Printf("Recommended: %s (Score: %.2f)\n", rec.ItemID, rec.Score)
}
```

### Feature Flags

```go
// Setup configuration
provider := ab.NewInMemoryConfigProvider()
configManager := ab.NewConfigManager(provider)

// Create feature flag
flag := &ab.FeatureFlag{
    ID:      "new_checkout",
    Name:    "New Checkout Flow",
    Enabled: true,
    Rules: []ab.FeatureFlagRule{
        {
            Conditions: []ab.RuleCondition{
                {
                    Attribute: "user_type",
                    Operator:  "equals",
                    Value:     "premium",
                },
            },
            Action: ab.RuleAction{Type: "enable"},
        },
    },
}

configManager.CreateFeatureFlag(ctx, flag)

// Evaluate flag for user
userAttrs := map[string]interface{}{"user_type": "premium"}
result, _ := configManager.EvaluateFeatureFlag(ctx, "new_checkout", "user123", userAttrs)

if result.Enabled {
    fmt.Println("Show new checkout flow")
}
```

## Advanced Features

### Statistical Analysis

The framework includes comprehensive statistical testing:

- **T-tests**: Compare means between variants
- **Confidence Intervals**: Calculate confidence intervals for metrics
- **Effect Size**: Measure practical significance of differences
- **Power Analysis**: Determine required sample sizes

### Quality Control

Built-in quality control mechanisms:

- **Sample Ratio Mismatch (SRM)**: Detect unexpected traffic distribution
- **Outlier Detection**: Identify and handle outliers in data
- **Bias Detection**: Check for biases across user segments
- **Novelty Effect**: Monitor for short-term novelty effects

### Analytics and Reporting

Comprehensive analytics capabilities:

- **Custom Metrics**: Define and track custom business metrics
- **Segmentation**: Analyze results across different user segments
- **Time Series**: Track metric trends over time
- **Export**: Export data in JSON or CSV formats

## Architecture

The framework is designed with modularity and extensibility in mind:

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Experiment    │  │     Bandit      │  │ Recommendation  │
│     Engine      │  │     Engine      │  │     Engine      │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               │
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Analytics     │  │     Config      │  │   Storage       │
│     Engine      │  │    Manager      │  │   Providers     │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Components

1. **Experiment Engine**: Core A/B testing functionality
2. **Bandit Engine**: Multi-armed bandit algorithms
3. **Recommendation Engine**: Personalization and recommendations
4. **Analytics Engine**: Statistical analysis and reporting
5. **Config Manager**: Feature flags and experiment configuration
6. **Storage Providers**: Pluggable storage backends

## Configuration

### Environment Variables

```bash
# Redis configuration (if using Redis provider)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Analytics settings
ANALYTICS_BATCH_SIZE=100
ANALYTICS_FLUSH_INTERVAL=60s

# Experiment settings
DEFAULT_CONFIDENCE_LEVEL=0.95
MIN_SAMPLE_SIZE=100
```

### Provider Configuration

The framework supports multiple storage providers:

- **In-Memory**: For testing and development
- **Redis**: For production with Redis backend
- **Database**: For SQL database storage
- **Custom**: Implement your own storage provider

## Best Practices

### A/B Testing

1. **Plan Your Experiments**: Define clear hypotheses and success metrics
2. **Sample Size Calculation**: Use power analysis to determine required sample sizes
3. **Randomization**: Ensure proper randomization to avoid bias
4. **Statistical Significance**: Wait for sufficient data before making decisions
5. **Multiple Testing**: Adjust for multiple comparisons when running multiple tests

### Feature Flags

1. **Gradual Rollouts**: Start with small percentages and gradually increase
2. **Monitoring**: Monitor key metrics during rollouts
3. **Rollback Plans**: Have clear rollback procedures for problematic releases
4. **Documentation**: Document flag purposes and cleanup schedules

### Recommendations

1. **Cold Start**: Handle new users and items gracefully
2. **Diversity**: Balance relevance with diversity in recommendations
3. **Freshness**: Include recent and trending content
4. **Feedback Loops**: Collect and incorporate user feedback

## Testing

Run the test suite:

```bash
go test ./...
```

Run specific tests:

```bash
go test -run TestExperimentEngine
go test -run TestRecommendationEngine
go test -run TestBanditEngine
```

## Performance Considerations

### Scalability

- **Horizontal Scaling**: All engines are stateless and can be scaled horizontally
- **Caching**: Built-in caching reduces database load
- **Batch Processing**: Analytics support batch processing for high-throughput scenarios

### Memory Usage

- **Efficient Data Structures**: Optimized for memory usage
- **Configurable Cache**: Adjustable cache sizes and TTL
- **Garbage Collection**: Minimal allocations to reduce GC pressure

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For questions and support:

- Create an issue on GitHub
- Check the documentation
- Review the test files for examples

## Roadmap

- [ ] Additional bandit algorithms (Contextual bandits)
- [ ] Deep learning-based recommendations
- [ ] Real-time streaming analytics
- [ ] Advanced statistical tests (Bayesian A/B testing)
- [ ] Mobile SDK support
- [ ] Dashboard and UI components
