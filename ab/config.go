package ab

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ConfigProvider defines interface for configuration providers
type ConfigProvider interface {
	GetConfig(ctx context.Context, key string) (interface{}, error)
	SetConfig(ctx context.Context, key string, value interface{}) error
	DeleteConfig(ctx context.Context, key string) error
	ListConfigs(ctx context.Context, prefix string) (map[string]interface{}, error)
	Watch(ctx context.Context, key string) (<-chan ConfigChange, error)
}

// ConfigChange represents a configuration change event
type ConfigChange struct {
	Key       string      `json:"key"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
}

// FeatureFlag represents a feature flag configuration
type FeatureFlag struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Rules       []FeatureFlagRule      `json:"rules"`
	Rollout     *RolloutConfig         `json:"rollout,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
}

// FeatureFlagRule defines targeting rules for feature flags
type FeatureFlagRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Conditions []RuleCondition        `json:"conditions"`
	Action     RuleAction             `json:"action"`
	Weight     float64                `json:"weight"`   // 0.0 to 1.0
	Priority   int                    `json:"priority"` // Higher number = higher priority
	Metadata   map[string]interface{} `json:"metadata"`
}

// RuleCondition defines a condition for feature flag rules
type RuleCondition struct {
	Attribute     string      `json:"attribute"`
	Operator      string      `json:"operator"` // "equals", "not_equals", "contains", "in", "not_in", "greater_than", "less_than"
	Value         interface{} `json:"value"`
	CaseSensitive bool        `json:"case_sensitive"`
}

// RuleAction defines the action to take when a rule matches
type RuleAction struct {
	Type   string                 `json:"type"`   // "enable", "disable", "variant", "redirect"
	Value  interface{}            `json:"value"`  // Variant ID, URL, etc.
	Config map[string]interface{} `json:"config"` // Additional configuration
}

// RolloutConfig defines gradual rollout configuration
type RolloutConfig struct {
	Strategy   string            `json:"strategy"`   // "percentage", "user_list", "attributes"
	Percentage float64           `json:"percentage"` // 0.0 to 100.0
	UserList   []string          `json:"user_list,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	StartTime  *time.Time        `json:"start_time,omitempty"`
	EndTime    *time.Time        `json:"end_time,omitempty"`
}

// ExperimentConfig represents configuration for an experiment
type ExperimentConfig struct {
	ExperimentID   string                  `json:"experiment_id"`
	TrafficSplit   map[string]float64      `json:"traffic_split"` // variant_id -> percentage
	TargetingRules []TargetingRule         `json:"targeting_rules"`
	SampleSize     *SampleSizeConfig       `json:"sample_size,omitempty"`
	StoppingRules  []StoppingRule          `json:"stopping_rules"`
	MetricConfig   map[string]MetricConfig `json:"metric_config"`
	QualityControl *QualityControlConfig   `json:"quality_control,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

// TargetingRule defines who should be included in an experiment
type TargetingRule struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Conditions []RuleCondition `json:"conditions"`
	Include    bool            `json:"include"` // true = include, false = exclude
	Priority   int             `json:"priority"`
}

// SampleSizeConfig defines sample size requirements
type SampleSizeConfig struct {
	MinSamplePerVariant int                  `json:"min_sample_per_variant"`
	MaxSamplePerVariant int                  `json:"max_sample_per_variant"`
	PowerAnalysis       *PowerAnalysisConfig `json:"power_analysis,omitempty"`
}

// PowerAnalysisConfig defines power analysis parameters
type PowerAnalysisConfig struct {
	Alpha                  float64 `json:"alpha"`                 // Type I error rate (e.g., 0.05)
	Beta                   float64 `json:"beta"`                  // Type II error rate (e.g., 0.2)
	MinDetectableEffect    float64 `json:"min_detectable_effect"` // Minimum effect size to detect
	BaselineConversionRate float64 `json:"baseline_conversion_rate"`
}

// StoppingRule defines when to stop an experiment
type StoppingRule struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`      // "significance", "sample_size", "time", "custom"
	Condition map[string]interface{} `json:"condition"` // Rule-specific parameters
	Action    string                 `json:"action"`    // "stop", "extend", "notify"
	Priority  int                    `json:"priority"`
}

// MetricConfig defines configuration for metrics
type MetricConfig struct {
	MetricID  string  `json:"metric_id"`
	IsPrimary bool    `json:"is_primary"`
	Weight    float64 `json:"weight"`    // For composite metrics
	Threshold float64 `json:"threshold"` // Alert threshold
	Direction string  `json:"direction"` // "increase", "decrease", "either"
}

// QualityControlConfig defines quality control parameters
type QualityControlConfig struct {
	SampleRatioMismatch *SRMConfig           `json:"sample_ratio_mismatch,omitempty"`
	OutlierDetection    *OutlierConfig       `json:"outlier_detection,omitempty"`
	NoveltyEffect       *NoveltyConfig       `json:"novelty_effect,omitempty"`
	BiasDetection       *BiasDetectionConfig `json:"bias_detection,omitempty"`
}

// SRMConfig defines Sample Ratio Mismatch detection
type SRMConfig struct {
	Enabled          bool    `json:"enabled"`
	TolerancePercent float64 `json:"tolerance_percent"` // Allowed deviation from expected ratio
	AlertThreshold   float64 `json:"alert_threshold"`   // P-value threshold for alerts
}

// OutlierConfig defines outlier detection parameters
type OutlierConfig struct {
	Enabled   bool    `json:"enabled"`
	Method    string  `json:"method"`    // "iqr", "zscore", "isolation_forest"
	Threshold float64 `json:"threshold"` // Method-specific threshold
	Action    string  `json:"action"`    // "exclude", "flag", "transform"
}

// NoveltyConfig defines novelty effect detection
type NoveltyConfig struct {
	Enabled        bool          `json:"enabled"`
	MonitorPeriod  time.Duration `json:"monitor_period"`  // How long to monitor for novelty
	BaselinePeriod time.Duration `json:"baseline_period"` // Period to establish baseline
}

// BiasDetectionConfig defines bias detection parameters
type BiasDetectionConfig struct {
	Enabled           bool     `json:"enabled"`
	CheckSegments     []string `json:"check_segments"` // Segments to check for bias
	SignificanceLevel float64  `json:"significance_level"`
}

// ConfigManager manages all configuration for A/B testing
type ConfigManager struct {
	provider     ConfigProvider
	featureFlags map[string]*FeatureFlag
	experiments  map[string]*ExperimentConfig
	cache        map[string]interface{}
	cacheTTL     time.Duration
	mu           sync.RWMutex
	watchers     map[string][]chan ConfigChange
	storage      ConfigStorage
	metrics      ConfigMetrics
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(provider ConfigProvider) *ConfigManager {
	return &ConfigManager{
		provider:     provider,
		featureFlags: make(map[string]*FeatureFlag),
		experiments:  make(map[string]*ExperimentConfig),
		cache:        make(map[string]interface{}),
		cacheTTL:     5 * time.Minute,
		watchers:     make(map[string][]chan ConfigChange),
		storage:      NewInMemoryConfigStorage(),
		metrics:      ConfigMetrics{},
	}
}

// NewConfigManagerWithStorage creates a new config manager with custom storage and metrics
func NewConfigManagerWithStorage(storage ConfigStorage, metrics *ConfigMetrics) *ConfigManager {
	if storage == nil {
		storage = NewInMemoryConfigStorage()
	}
	if metrics == nil {
		metrics = &ConfigMetrics{}
	}
	return &ConfigManager{
		storage: storage,
		metrics: *metrics,
	}
}

// ConfigStorage defines interface for persisting feature flags and experiment configs
// In production, implement with Redis, SQL, etc.
type ConfigStorage interface {
	SaveFeatureFlag(flag *FeatureFlag) error
	SaveExperimentConfig(config *ExperimentConfig) error
	ListFeatureFlags() ([]*FeatureFlag, error)
	ListExperimentConfigs() ([]*ExperimentConfig, error)
}

// InMemoryConfigStorage is a simple in-memory implementation
type InMemoryConfigStorage struct {
	flags       map[string]*FeatureFlag
	experiments map[string]*ExperimentConfig
}

func NewInMemoryConfigStorage() *InMemoryConfigStorage {
	return &InMemoryConfigStorage{
		flags:       make(map[string]*FeatureFlag),
		experiments: make(map[string]*ExperimentConfig),
	}
}

func (s *InMemoryConfigStorage) SaveFeatureFlag(flag *FeatureFlag) error {
	s.flags[flag.ID] = flag
	return nil
}
func (s *InMemoryConfigStorage) SaveExperimentConfig(config *ExperimentConfig) error {
	s.experiments[config.ExperimentID] = config
	return nil
}
func (s *InMemoryConfigStorage) ListFeatureFlags() ([]*FeatureFlag, error) {
	var out []*FeatureFlag
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out, nil
}
func (s *InMemoryConfigStorage) ListExperimentConfigs() ([]*ExperimentConfig, error) {
	var out []*ExperimentConfig
	for _, c := range s.experiments {
		out = append(out, c)
	}
	return out, nil
}

// ConfigMetrics tracks operational metrics for config manager
type ConfigMetrics struct {
	FeatureFlagEvaluations int64
	FeatureFlagErrors      int64
	ConfigChanges          int64
}

// CreateFeatureFlag creates a new feature flag
func (c *ConfigManager) CreateFeatureFlag(ctx context.Context, flag *FeatureFlag) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if flag.ID == "" {
		flag.ID = generateID()
	}

	flag.CreatedAt = time.Now()
	flag.UpdatedAt = time.Now()

	// Store in provider
	key := fmt.Sprintf("feature_flags/%s", flag.ID)
	if err := c.provider.SetConfig(ctx, key, flag); err != nil {
		c.metrics.FeatureFlagErrors++
		return err
	}

	// Update local cache
	c.featureFlags[flag.ID] = flag
	_ = c.storage.SaveFeatureFlag(flag)
	c.metrics.ConfigChanges++

	return nil
}

// GetFeatureFlag retrieves a feature flag
func (c *ConfigManager) GetFeatureFlag(ctx context.Context, flagID string) (*FeatureFlag, error) {
	c.mu.RLock()
	if flag, exists := c.featureFlags[flagID]; exists {
		c.mu.RUnlock()
		return flag, nil
	}
	c.mu.RUnlock()

	// Load from provider
	key := fmt.Sprintf("feature_flags/%s", flagID)
	value, err := c.provider.GetConfig(ctx, key)
	if err != nil {
		return nil, err
	}

	flag := &FeatureFlag{}
	if err := c.unmarshalConfig(value, flag); err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.featureFlags[flagID] = flag
	c.mu.Unlock()

	return flag, nil
}

// EvaluateFeatureFlag evaluates a feature flag for a user
func (c *ConfigManager) EvaluateFeatureFlag(ctx context.Context, flagID, userID string, userAttributes map[string]interface{}) (*FeatureFlagResult, error) {
	flag, err := c.GetFeatureFlag(ctx, flagID)
	if err != nil {
		c.metrics.FeatureFlagErrors++
		return nil, err
	}

	result := &FeatureFlagResult{
		FlagID:      flagID,
		UserID:      userID,
		Enabled:     false,
		Variant:     "",
		Config:      make(map[string]interface{}),
		RuleMatched: "",
		EvaluatedAt: time.Now(),
	}

	// Check if flag is globally enabled
	if !flag.Enabled {
		result.Reason = "flag_disabled"
		c.metrics.FeatureFlagEvaluations++
		return result, nil
	}

	// Check rollout configuration
	if flag.Rollout != nil && !c.evaluateRollout(flag.Rollout, userID, userAttributes) {
		result.Reason = "not_in_rollout"
		c.metrics.FeatureFlagEvaluations++
		return result, nil
	}

	// Evaluate rules in priority order
	sort.Slice(flag.Rules, func(i, j int) bool {
		return flag.Rules[i].Priority > flag.Rules[j].Priority
	})

	for _, rule := range flag.Rules {
		if c.evaluateRule(rule, userAttributes) {
			result.Enabled = true
			result.RuleMatched = rule.ID
			result.Reason = "rule_matched"

			// Apply rule action
			switch rule.Action.Type {
			case "enable":
				result.Enabled = true
			case "disable":
				result.Enabled = false
			case "variant":
				if variant, ok := rule.Action.Value.(string); ok {
					result.Variant = variant
				}
			}

			// Apply rule config
			for k, v := range rule.Action.Config {
				result.Config[k] = v
			}

			c.metrics.FeatureFlagEvaluations++
			return result, nil
		}
	}

	// Default behavior if no rules match
	result.Enabled = flag.Enabled
	result.Reason = "default"
	c.metrics.FeatureFlagEvaluations++

	return result, nil
}

// FeatureFlagResult represents the result of feature flag evaluation
type FeatureFlagResult struct {
	FlagID      string                 `json:"flag_id"`
	UserID      string                 `json:"user_id"`
	Enabled     bool                   `json:"enabled"`
	Variant     string                 `json:"variant"`
	Config      map[string]interface{} `json:"config"`
	RuleMatched string                 `json:"rule_matched"`
	Reason      string                 `json:"reason"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
}

// evaluateRollout checks if a user should be included in a rollout
func (c *ConfigManager) evaluateRollout(rollout *RolloutConfig, userID string, userAttributes map[string]interface{}) bool {
	switch rollout.Strategy {
	case "percentage":
		return Rollout(userID, uint64(rollout.Percentage))
	case "user_list":
		for _, user := range rollout.UserList {
			if user == userID {
				return true
			}
		}
		return false
	case "attributes":
		for key, expectedValue := range rollout.Attributes {
			if actualValue, exists := userAttributes[key]; !exists || fmt.Sprintf("%v", actualValue) != expectedValue {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// evaluateRule checks if a rule matches the user attributes
func (c *ConfigManager) evaluateRule(rule FeatureFlagRule, userAttributes map[string]interface{}) bool {
	for _, condition := range rule.Conditions {
		if !c.evaluateCondition(condition, userAttributes) {
			return false
		}
	}
	return true
}

// evaluateCondition checks if a condition matches
func (c *ConfigManager) evaluateCondition(condition RuleCondition, userAttributes map[string]interface{}) bool {
	actualValue, exists := userAttributes[condition.Attribute]
	if !exists {
		return false
	}

	actualStr := fmt.Sprintf("%v", actualValue)
	expectedStr := fmt.Sprintf("%v", condition.Value)

	if !condition.CaseSensitive {
		actualStr = fmt.Sprintf("%v", actualValue) // Would need proper case conversion
		expectedStr = fmt.Sprintf("%v", condition.Value)
	}

	switch condition.Operator {
	case "equals":
		return actualStr == expectedStr
	case "not_equals":
		return actualStr != expectedStr
	case "contains":
		return fmt.Sprintf("%v", actualValue) != "" && fmt.Sprintf("%v", condition.Value) != ""
	case "in":
		if list, ok := condition.Value.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", item) == actualStr {
					return true
				}
			}
		}
		return false
	case "not_in":
		if list, ok := condition.Value.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", item) == actualStr {
					return false
				}
			}
			return true
		}
		return true
	default:
		return false
	}
}

// CreateExperimentConfig creates experiment configuration
func (c *ConfigManager) CreateExperimentConfig(ctx context.Context, config *ExperimentConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	// Validate traffic split
	totalSplit := 0.0
	for _, percentage := range config.TrafficSplit {
		totalSplit += percentage
	}
	if math.Abs(totalSplit-100.0) > 0.001 {
		return fmt.Errorf("traffic split must sum to 100%%, got %.2f%%", totalSplit)
	}

	// Store in provider
	key := fmt.Sprintf("experiments/%s", config.ExperimentID)
	if err := c.provider.SetConfig(ctx, key, config); err != nil {
		c.metrics.FeatureFlagErrors++
		return err
	}

	// Update local cache
	c.experiments[config.ExperimentID] = config
	_ = c.storage.SaveExperimentConfig(config)
	c.metrics.ConfigChanges++

	return nil
}

// GetExperimentConfig retrieves experiment configuration
func (c *ConfigManager) GetExperimentConfig(ctx context.Context, experimentID string) (*ExperimentConfig, error) {
	c.mu.RLock()
	if config, exists := c.experiments[experimentID]; exists {
		c.mu.RUnlock()
		return config, nil
	}
	c.mu.RUnlock()

	// Load from provider
	key := fmt.Sprintf("experiments/%s", experimentID)
	value, err := c.provider.GetConfig(ctx, key)
	if err != nil {
		return nil, err
	}

	config := &ExperimentConfig{}
	if err := c.unmarshalConfig(value, config); err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.experiments[experimentID] = config
	c.mu.Unlock()

	return config, nil
}

// WatchConfig watches for configuration changes
func (c *ConfigManager) WatchConfig(ctx context.Context, key string) (<-chan ConfigChange, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create channel for this watcher
	ch := make(chan ConfigChange, 10)
	c.watchers[key] = append(c.watchers[key], ch)

	// Start watching from provider
	providerCh, err := c.provider.Watch(ctx, key)
	if err != nil {
		return nil, err
	}

	// Forward changes to all watchers
	go func() {
		for change := range providerCh {
			c.mu.RLock()
			watchers := c.watchers[key]
			c.mu.RUnlock()

			for _, watcher := range watchers {
				select {
				case watcher <- change:
				case <-ctx.Done():
					return
				default:
					// Channel full, skip this change
				}
			}
		}
	}()

	return ch, nil
}

// ValidateConfig validates configuration before saving
func (c *ConfigManager) ValidateConfig(config interface{}) error {
	switch cfg := config.(type) {
	case *FeatureFlag:
		return c.validateFeatureFlag(cfg)
	case *ExperimentConfig:
		return c.validateExperimentConfig(cfg)
	default:
		return fmt.Errorf("unsupported config type: %T", config)
	}
}

// validateFeatureFlag validates feature flag configuration
func (c *ConfigManager) validateFeatureFlag(flag *FeatureFlag) error {
	if flag.ID == "" {
		return fmt.Errorf("feature flag ID is required")
	}
	if flag.Name == "" {
		return fmt.Errorf("feature flag name is required")
	}

	// Validate rules
	for i, rule := range flag.Rules {
		if rule.Weight < 0 || rule.Weight > 1 {
			return fmt.Errorf("rule %d: weight must be between 0 and 1", i)
		}

		for j, condition := range rule.Conditions {
			if condition.Attribute == "" {
				return fmt.Errorf("rule %d, condition %d: attribute is required", i, j)
			}
			if condition.Operator == "" {
				return fmt.Errorf("rule %d, condition %d: operator is required", i, j)
			}
		}
	}

	return nil
}

// validateExperimentConfig validates experiment configuration
func (c *ConfigManager) validateExperimentConfig(config *ExperimentConfig) error {
	if config.ExperimentID == "" {
		return fmt.Errorf("experiment ID is required")
	}

	// Validate traffic split
	totalSplit := 0.0
	for variant, percentage := range config.TrafficSplit {
		if variant == "" {
			return fmt.Errorf("variant ID cannot be empty")
		}
		if percentage < 0 || percentage > 100 {
			return fmt.Errorf("traffic percentage for variant %s must be between 0 and 100", variant)
		}
		totalSplit += percentage
	}

	if math.Abs(totalSplit-100.0) > 0.001 {
		return fmt.Errorf("traffic split must sum to 100%%, got %.2f%%", totalSplit)
	}

	return nil
}

// unmarshalConfig unmarshals configuration from provider format
func (c *ConfigManager) unmarshalConfig(value interface{}, target interface{}) error {
	// Convert to JSON and back to handle different provider formats
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}

// InMemoryConfigProvider provides an in-memory configuration store
type InMemoryConfigProvider struct {
	configs  map[string]interface{}
	watchers map[string][]chan ConfigChange
	mu       sync.RWMutex
}

// NewInMemoryConfigProvider creates a new in-memory config provider
func NewInMemoryConfigProvider() *InMemoryConfigProvider {
	return &InMemoryConfigProvider{
		configs:  make(map[string]interface{}),
		watchers: make(map[string][]chan ConfigChange),
	}
}

// GetConfig retrieves a configuration value
func (p *InMemoryConfigProvider) GetConfig(ctx context.Context, key string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	value, exists := p.configs[key]
	if !exists {
		return nil, fmt.Errorf("config key %s not found", key)
	}

	return value, nil
}

// SetConfig sets a configuration value
func (p *InMemoryConfigProvider) SetConfig(ctx context.Context, key string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldValue := p.configs[key]
	p.configs[key] = value

	// Notify watchers
	change := ConfigChange{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  value,
		Timestamp: time.Now(),
		Source:    "in_memory",
	}

	for _, ch := range p.watchers[key] {
		select {
		case ch <- change:
		default:
			// Channel full, skip
		}
	}

	return nil
}

// DeleteConfig deletes a configuration value
func (p *InMemoryConfigProvider) DeleteConfig(ctx context.Context, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldValue := p.configs[key]
	delete(p.configs, key)

	// Notify watchers
	change := ConfigChange{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  nil,
		Timestamp: time.Now(),
		Source:    "in_memory",
	}

	for _, ch := range p.watchers[key] {
		select {
		case ch <- change:
		default:
			// Channel full, skip
		}
	}

	return nil
}

// ListConfigs lists all configurations with a given prefix
func (p *InMemoryConfigProvider) ListConfigs(ctx context.Context, prefix string) (map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]interface{})
	for key, value := range p.configs {
		if len(prefix) == 0 || fmt.Sprintf("%s", key)[:len(prefix)] == prefix {
			result[key] = value
		}
	}

	return result, nil
}

// Watch watches for changes to a configuration key
func (p *InMemoryConfigProvider) Watch(ctx context.Context, key string) (<-chan ConfigChange, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan ConfigChange, 10)
	p.watchers[key] = append(p.watchers[key], ch)

	return ch, nil
}

// ListFeatureFlags returns all feature flags
func (c *ConfigManager) ListFeatureFlags(ctx context.Context) ([]*FeatureFlag, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	flags := make([]*FeatureFlag, 0, len(c.featureFlags))
	for _, f := range c.featureFlags {
		flags = append(flags, f)
	}
	return flags, nil
}

// ListExperimentConfigs returns all experiment configs
func (c *ConfigManager) ListExperimentConfigs(ctx context.Context) ([]*ExperimentConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	configs := make([]*ExperimentConfig, 0, len(c.experiments))
	for _, cfg := range c.experiments {
		configs = append(configs, cfg)
	}
	return configs, nil
}

// DeleteFeatureFlag deletes a feature flag by ID
func (c *ConfigManager) DeleteFeatureFlag(ctx context.Context, flagID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.featureFlags, flagID)
	c.metrics.ConfigChanges++
	return c.provider.DeleteConfig(ctx, "feature_flags/"+flagID)
}

// DeleteExperimentConfig deletes an experiment config by ID
func (c *ConfigManager) DeleteExperimentConfig(ctx context.Context, experimentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.experiments, experimentID)
	c.metrics.ConfigChanges++
	return c.provider.DeleteConfig(ctx, "experiments/"+experimentID)
}

// GetMetrics returns config manager metrics
func (c *ConfigManager) GetMetrics() ConfigMetrics {
	return c.metrics
}
