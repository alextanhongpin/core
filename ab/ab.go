package ab

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"time"

	"github.com/spaolacci/murmur3"
)

// Hash generates a consistent hash for a given key within the specified size
func Hash(key string, size uint64) uint64 {
	return murmur3.Sum64([]byte(key)) % size
}

// Rollout determines if a user should be included in a rollout based on percentage
func Rollout(key string, percentage uint64) bool {
	return percentage > 0 && Hash(key, 100) < percentage
}

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	StatusDraft    ExperimentStatus = "draft"
	StatusActive   ExperimentStatus = "active"
	StatusPaused   ExperimentStatus = "paused"
	StatusComplete ExperimentStatus = "complete"
)

// Variant represents a single variant in an A/B test
type Variant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Weight      float64                `json:"weight"`      // 0.0 to 1.0
	Config      map[string]interface{} `json:"config"`      // Variant-specific configuration
	Conversions int64                  `json:"conversions"` // Number of conversions
	Impressions int64                  `json:"impressions"` // Number of impressions
}

// Experiment represents an A/B test experiment
type Experiment struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      ExperimentStatus `json:"status"`
	Variants    []Variant        `json:"variants"`
	StartTime   *time.Time       `json:"start_time,omitempty"`
	EndTime     *time.Time       `json:"end_time,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// Targeting criteria
	TargetPercentage float64           `json:"target_percentage"` // 0.0 to 1.0
	Filters          map[string]string `json:"filters"`           // User attributes for targeting

	// Analytics
	MinSampleSize       int     `json:"min_sample_size"`
	ConfidenceLevel     float64 `json:"confidence_level"` // e.g., 0.95 for 95%
	MinDetectableEffect float64 `json:"min_detectable_effect"`
}

// Assignment represents a user's assignment to an experiment variant
type Assignment struct {
	UserID       string            `json:"user_id"`
	ExperimentID string            `json:"experiment_id"`
	VariantID    string            `json:"variant_id"`
	AssignedAt   time.Time         `json:"assigned_at"`
	Attributes   map[string]string `json:"attributes"`
}

// ConversionEvent represents a conversion event for analytics
type ConversionEvent struct {
	UserID       string                 `json:"user_id"`
	ExperimentID string                 `json:"experiment_id"`
	VariantID    string                 `json:"variant_id"`
	EventType    string                 `json:"event_type"`
	Value        float64                `json:"value,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ExperimentEngine manages A/B test experiments
type ExperimentEngine struct {
	experiments map[string]*Experiment
	assignments map[string]*Assignment // key: userID_experimentID
	store       ExperimentStore
}

// NewExperimentEngine creates a new experiment engine
func NewExperimentEngine(store ExperimentStore) *ExperimentEngine {
	return &ExperimentEngine{
		experiments: make(map[string]*Experiment),
		assignments: make(map[string]*Assignment),
		store:       store,
	}
}

// CreateExperiment creates a new experiment
func (e *ExperimentEngine) CreateExperiment(exp *Experiment) error {
	if exp.ID == "" {
		exp.ID = generateID()
	}

	// Validate variants weights sum to 1.0
	totalWeight := 0.0
	for _, variant := range exp.Variants {
		totalWeight += variant.Weight
	}
	if math.Abs(totalWeight-1.0) > 0.001 {
		return fmt.Errorf("variant weights must sum to 1.0, got %f", totalWeight)
	}

	exp.CreatedAt = time.Now()
	exp.UpdatedAt = time.Now()
	e.experiments[exp.ID] = exp

	// Save to store
	if e.store != nil {
		if err := e.store.SaveExperiment(exp); err != nil {
			return fmt.Errorf("failed to save experiment to store: %w", err)
		}
	}

	return nil
}

// AssignVariant assigns a user to a variant in an experiment
func (e *ExperimentEngine) AssignVariant(ctx context.Context, userID, experimentID string, userAttributes map[string]string) (*Assignment, error) {
	exp, exists := e.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	if exp.Status != StatusActive {
		return nil, fmt.Errorf("experiment %s is not active", experimentID)
	}

	// Check if user already has an assignment
	assignmentKey := fmt.Sprintf("%s_%s", userID, experimentID)
	if assignment, exists := e.assignments[assignmentKey]; exists {
		return assignment, nil
	}

	// Check if user should be included in experiment
	if !e.shouldIncludeUser(userID, exp, userAttributes) {
		return nil, nil
	}

	// Assign variant based on hash and weights
	variantID := e.selectVariant(userID, experimentID, exp.Variants)

	assignment := &Assignment{
		UserID:       userID,
		ExperimentID: experimentID,
		VariantID:    variantID,
		AssignedAt:   time.Now(),
		Attributes:   userAttributes,
	}

	e.assignments[assignmentKey] = assignment

	// Increment impressions for the variant
	for i := range exp.Variants {
		if exp.Variants[i].ID == variantID {
			exp.Variants[i].Impressions++
			break
		}
	}

	// Save assignment to store
	if e.store != nil {
		if err := e.store.SaveAssignment(assignment); err != nil {
			return nil, fmt.Errorf("failed to save assignment to store: %w", err)
		}
	}

	return assignment, nil
}

// shouldIncludeUser determines if a user should be included in the experiment
func (e *ExperimentEngine) shouldIncludeUser(userID string, exp *Experiment, userAttributes map[string]string) bool {
	// Check target percentage
	if !Rollout(userID+exp.ID, uint64(exp.TargetPercentage*100)) {
		return false
	}

	// Check filters
	for key, expectedValue := range exp.Filters {
		if actualValue, exists := userAttributes[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// selectVariant selects a variant based on consistent hashing and weights
func (e *ExperimentEngine) selectVariant(userID, experimentID string, variants []Variant) string {
	// Create cumulative weight distribution
	cumulativeWeights := make([]float64, len(variants))
	cumulativeWeights[0] = variants[0].Weight
	for i := 1; i < len(variants); i++ {
		cumulativeWeights[i] = cumulativeWeights[i-1] + variants[i].Weight
	}

	// Generate consistent random value between 0 and 1
	hashValue := Hash(userID+experimentID, 1000000)
	randomValue := float64(hashValue) / 1000000.0

	// Find the variant based on cumulative weights
	for i, weight := range cumulativeWeights {
		if randomValue <= weight {
			return variants[i].ID
		}
	}

	// Fallback to first variant
	return variants[0].ID
}

// RecordConversion records a conversion event for analytics
func (e *ExperimentEngine) RecordConversion(ctx context.Context, event *ConversionEvent) error {
	exp, exists := e.experiments[event.ExperimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", event.ExperimentID)
	}

	// Update conversion count for the variant
	for i := range exp.Variants {
		if exp.Variants[i].ID == event.VariantID {
			exp.Variants[i].Conversions++
			break
		}
	}

	// Save conversion event to store
	if e.store != nil {
		if err := e.store.SaveConversion(event); err != nil {
			return fmt.Errorf("failed to save conversion event to store: %w", err)
		}
	}

	return nil
}

// GetExperimentResults calculates statistical results for an experiment
func (e *ExperimentEngine) GetExperimentResults(experimentID string) (*ExperimentResults, error) {
	exp, exists := e.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	results := &ExperimentResults{
		ExperimentID: experimentID,
		Variants:     make([]VariantResults, len(exp.Variants)),
	}

	for i, variant := range exp.Variants {
		conversionRate := 0.0
		if variant.Impressions > 0 {
			conversionRate = float64(variant.Conversions) / float64(variant.Impressions)
		}

		results.Variants[i] = VariantResults{
			VariantID:      variant.ID,
			Impressions:    variant.Impressions,
			Conversions:    variant.Conversions,
			ConversionRate: conversionRate,
			ConfidenceInterval: e.calculateConfidenceInterval(
				variant.Conversions, variant.Impressions, exp.ConfidenceLevel,
			),
		}
	}

	// Calculate statistical significance between variants
	if len(results.Variants) >= 2 {
		control := results.Variants[0]
		for i := 1; i < len(results.Variants); i++ {
			treatment := results.Variants[i]
			pValue, _ := e.calculateZTest(control, treatment)
			results.Variants[i].PValue = pValue
			results.Variants[i].IsSignificant = pValue < (1.0 - exp.ConfidenceLevel)
		}
	}

	return results, nil
}

// ExperimentResults contains the statistical results of an experiment
type ExperimentResults struct {
	ExperimentID string           `json:"experiment_id"`
	Variants     []VariantResults `json:"variants"`
}

// VariantResults contains results for a single variant
type VariantResults struct {
	VariantID          string             `json:"variant_id"`
	Impressions        int64              `json:"impressions"`
	Conversions        int64              `json:"conversions"`
	ConversionRate     float64            `json:"conversion_rate"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	PValue             float64            `json:"p_value,omitempty"`
	IsSignificant      bool               `json:"is_significant"`
}

// ConfidenceInterval represents a confidence interval
type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// calculateConfidenceInterval calculates confidence interval for conversion rate
func (e *ExperimentEngine) calculateConfidenceInterval(conversions, impressions int64, confidenceLevel float64) ConfidenceInterval {
	if impressions == 0 {
		return ConfidenceInterval{Lower: 0, Upper: 0}
	}

	p := float64(conversions) / float64(impressions)
	n := float64(impressions)

	// Z-score for confidence level (approximation for 95%)
	z := 1.96
	if confidenceLevel == 0.99 {
		z = 2.576
	} else if confidenceLevel == 0.90 {
		z = 1.645
	}

	margin := z * math.Sqrt(p*(1-p)/n)

	return ConfidenceInterval{
		Lower: math.Max(0, p-margin),
		Upper: math.Min(1, p+margin),
	}
}

// calculateZTest performs a two-proportion z-test
func (e *ExperimentEngine) calculateZTest(control, treatment VariantResults) (float64, error) {
	if control.Impressions == 0 || treatment.Impressions == 0 {
		return 1.0, fmt.Errorf("insufficient data for statistical test")
	}

	p1 := control.ConversionRate
	p2 := treatment.ConversionRate
	n1 := float64(control.Impressions)
	n2 := float64(treatment.Impressions)

	// Pooled proportion
	pooledP := (float64(control.Conversions) + float64(treatment.Conversions)) / (n1 + n2)

	// Standard error
	se := math.Sqrt(pooledP * (1 - pooledP) * (1/n1 + 1/n2))

	if se == 0 {
		return 1.0, fmt.Errorf("standard error is zero")
	}

	// Z-score
	z := (p2 - p1) / se

	// Two-tailed p-value (approximation)
	pValue := 2 * (1 - math.Abs(z)/2.0) // Simplified approximation

	return pValue, nil
}

// ExperimentStore defines the interface for persisting experiments, variants, assignments, and conversions
// In production, implement with Redis, SQL, etc.
type ExperimentStore interface {
	SaveExperiment(exp *Experiment) error
	GetExperiment(id string) (*Experiment, error)
	ListExperiments() ([]*Experiment, error)
	SaveAssignment(assignment *Assignment) error
	GetAssignment(userID, experimentID string) (*Assignment, error)
	ListAssignments(experimentID string) ([]*Assignment, error)
	SaveConversion(event *ConversionEvent) error
	ListConversions(experimentID string) ([]*ConversionEvent, error)
}

// InMemoryExperimentStore is a simple in-memory implementation
// (for testing and development)
type InMemoryExperimentStore struct {
	experiments map[string]*Experiment
	assignments map[string]*Assignment
	conversions map[string][]*ConversionEvent // experimentID -> events
}

func NewInMemoryExperimentStore() *InMemoryExperimentStore {
	return &InMemoryExperimentStore{
		experiments: make(map[string]*Experiment),
		assignments: make(map[string]*Assignment),
		conversions: make(map[string][]*ConversionEvent),
	}
}

func (s *InMemoryExperimentStore) SaveExperiment(exp *Experiment) error {
	s.experiments[exp.ID] = exp
	return nil
}
func (s *InMemoryExperimentStore) GetExperiment(id string) (*Experiment, error) {
	exp, ok := s.experiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment %s not found", id)
	}
	return exp, nil
}
func (s *InMemoryExperimentStore) ListExperiments() ([]*Experiment, error) {
	var exps []*Experiment
	for _, exp := range s.experiments {
		exps = append(exps, exp)
	}
	return exps, nil
}
func (s *InMemoryExperimentStore) SaveAssignment(assignment *Assignment) error {
	key := fmt.Sprintf("%s_%s", assignment.UserID, assignment.ExperimentID)
	s.assignments[key] = assignment
	return nil
}
func (s *InMemoryExperimentStore) GetAssignment(userID, experimentID string) (*Assignment, error) {
	key := fmt.Sprintf("%s_%s", userID, experimentID)
	assignment, ok := s.assignments[key]
	if !ok {
		return nil, fmt.Errorf("assignment not found")
	}
	return assignment, nil
}
func (s *InMemoryExperimentStore) ListAssignments(experimentID string) ([]*Assignment, error) {
	var result []*Assignment
	for _, a := range s.assignments {
		if a.ExperimentID == experimentID {
			result = append(result, a)
		}
	}
	return result, nil
}
func (s *InMemoryExperimentStore) SaveConversion(event *ConversionEvent) error {
	s.conversions[event.ExperimentID] = append(s.conversions[event.ExperimentID], event)
	return nil
}
func (s *InMemoryExperimentStore) ListConversions(experimentID string) ([]*ConversionEvent, error) {
	return s.conversions[experimentID], nil
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return fmt.Sprintf("exp_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("exp_%x", bytes)
}
