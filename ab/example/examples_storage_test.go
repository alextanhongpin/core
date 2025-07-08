package main_test

import (
	"context"
	"fmt"

	// TODO: Replace the following import path with your actual module path as defined in go.mod
	"github.com/alextanhongpin/core/ab"
)

// --- Analytics Storage Example ---
type CustomAnalyticsStorage struct{}

func (s *CustomAnalyticsStorage) SaveMetricValue(ctx context.Context, value ab.MetricValue) error {
	// Implement persistent save logic here
	fmt.Println("[Analytics] SaveMetricValue:", value)
	return nil
}
func (s *CustomAnalyticsStorage) GetMetricValues(ctx context.Context, experimentID string, timeRange ab.TimeRange) ([]ab.MetricValue, error) {
	// Implement persistent fetch logic here
	return nil, nil
}
func (s *CustomAnalyticsStorage) SaveSegment(ctx context.Context, segment *ab.Segment) error {
	fmt.Println("[Analytics] SaveSegment:", segment)
	return nil
}
func (s *CustomAnalyticsStorage) GetSegments(ctx context.Context) ([]*ab.Segment, error) {
	return nil, nil
}

// --- Bandit Storage Example ---
type CustomBanditStorage struct{}

func (s *CustomBanditStorage) SaveExperiment(exp *ab.BanditExperiment) error {
	fmt.Println("[Bandit] SaveExperiment:", exp)
	return nil
}
func (s *CustomBanditStorage) SaveArm(expID string, arm *ab.BanditArm) error {
	fmt.Println("[Bandit] SaveArm:", expID, arm)
	return nil
}
func (s *CustomBanditStorage) SaveResult(result ab.BanditResult) error {
	fmt.Println("[Bandit] SaveResult:", result)
	return nil
}
func (s *CustomBanditStorage) ListResults(expID string, limit int) ([]ab.BanditResult, error) {
	return nil, nil
}

// --- Recommendation Storage Example ---
type CustomRecommendationStorage struct{}

func (s *CustomRecommendationStorage) SaveRecommendation(rec ab.Recommendation) error {
	fmt.Println("[Recommendation] SaveRecommendation:", rec)
	return nil
}
func (s *CustomRecommendationStorage) SaveInteraction(interaction ab.Interaction) error {
	fmt.Println("[Recommendation] SaveInteraction:", interaction)
	return nil
}
func (s *CustomRecommendationStorage) SaveItem(item *ab.Item) error {
	fmt.Println("[Recommendation] SaveItem:", item)
	return nil
}
func (s *CustomRecommendationStorage) SaveUser(user *ab.User) error {
	fmt.Println("[Recommendation] SaveUser:", user)
	return nil
}
func (s *CustomRecommendationStorage) ListRecommendations(userID string, limit int) ([]ab.Recommendation, error) {
	return nil, nil
}

// --- Config Storage Example ---
type CustomConfigStorage struct{}

func (s *CustomConfigStorage) SaveConfig(config ab.ExperimentConfig) error {
	fmt.Println("[Config] SaveConfig:", config)
	return nil
}
func (s *CustomConfigStorage) SaveExperimentConfig(config *ab.ExperimentConfig) error {
	fmt.Println("[Config] SaveExperimentConfig:", config)
	return nil
}
func (s *CustomConfigStorage) GetConfig(experimentID string) (*ab.ExperimentConfig, error) {
	return nil, nil
}
func (s *CustomConfigStorage) ListExperimentConfigs() ([]*ab.ExperimentConfig, error) {
	return nil, nil
}
func (s *CustomConfigStorage) ListFeatureFlags() ([]*ab.FeatureFlag, error) {
	return nil, nil
}
func (s *CustomConfigStorage) SaveFeatureFlag(flag *ab.FeatureFlag) error {
	fmt.Println("[Config] SaveFeatureFlag:", flag)
	return nil
}

// Example functions to demonstrate plugging in custom storage
func ExampleAnalyticsEngine_withCustomStorage() {
	analytics := ab.NewAnalyticsEngineWithStorage(&CustomAnalyticsStorage{}, nil)
	fmt.Println("Custom analytics storage plugged in:", analytics != nil)

	// Output:
	// Custom analytics storage plugged in: true
}

func ExampleBanditEngine_withCustomStorage() {
	bandit := ab.NewBanditEngineWithStorage(&CustomBanditStorage{}, nil)
	fmt.Println("Custom bandit storage plugged in:", bandit != nil)

	// Output:
	// Custom bandit storage plugged in: true
}

func ExampleRecommendationEngine_withCustomStorage() {
	recommendation := ab.NewRecommendationEngineWithStorage(&CustomRecommendationStorage{}, nil)
	fmt.Println("Custom recommendation storage plugged in:", recommendation != nil)

	// Output:
	// Custom recommendation storage plugged in: true
}

func ExampleConfigManager_withCustomStorage() {
	config := ab.NewConfigManagerWithStorage(&CustomConfigStorage{}, nil)
	fmt.Println("Custom config storage plugged in:", config != nil)

	// Output:
	// Custom config storage plugged in: true
}
