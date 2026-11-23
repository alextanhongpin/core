package ab

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/spaolacci/murmur3"
)

// Hash returns a stable bucket in [0, size) for key. It panics if size is 0.
func Hash(key string, size uint64) uint64 {
	return murmur3.Sum64([]byte(key)) % size
}

// Rollout reports whether key falls in the first percentage (0-100) of 100
// stable buckets.
func Rollout(key string, percentage uint64) bool {
	return percentage > 0 && Hash(key, 100) < percentage
}

// rolloutFraction is Rollout for a fraction (0-1), rounded to the nearest
// percent. Rounding, rather than truncating, keeps fractions such as 0.29
// (0.29*100 == 28.999...) from becoming 28%.
func rolloutFraction(key string, fraction float64) bool {
	return Rollout(key, uint64(math.Round(fraction*100)))
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
	Conversions int64                  `json:"conversions"` // Number of converted users
	Impressions int64                  `json:"impressions"` // Number of assigned users
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
	TargetPercentage float64           `json:"target_percentage"` // (0.0, 1.0]
	Filters          map[string]string `json:"filters"`           // User attributes for targeting

	// Analytics
	MinSampleSize       int     `json:"min_sample_size"`
	ConfidenceLevel     float64 `json:"confidence_level"` // e.g., 0.95 for 95%; defaults to 0.95
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

type assignmentKey struct {
	userID, experimentID string
}

// ExperimentEngine assigns users to weighted variants and counts unique
// assigned and converted users per variant. It is safe for concurrent use.
// State is kept in memory; the store, if any, only receives copies of
// experiments, assignments and conversion events.
type ExperimentEngine struct {
	mu          sync.Mutex
	experiments map[string]*Experiment
	assignments map[assignmentKey]*Assignment
	converted   map[assignmentKey]bool
	store       ExperimentStore
}

// NewExperimentEngine creates a new experiment engine. store may be nil.
func NewExperimentEngine(store ExperimentStore) *ExperimentEngine {
	return &ExperimentEngine{
		experiments: make(map[string]*Experiment),
		assignments: make(map[assignmentKey]*Assignment),
		converted:   make(map[assignmentKey]bool),
		store:       store,
	}
}

// CreateExperiment validates and registers exp. The engine takes ownership
// of exp: callers must not modify it afterwards. Variants need unique,
// non-empty IDs and non-negative weights summing to 1, and TargetPercentage
// must be in (0, 1].
func (e *ExperimentEngine) CreateExperiment(exp *Experiment) error {
	if exp == nil {
		return errors.New("ab: experiment is nil")
	}
	if len(exp.Variants) == 0 {
		return errors.New("ab: experiment needs at least one variant")
	}
	if !(exp.TargetPercentage > 0 && exp.TargetPercentage <= 1) {
		return fmt.Errorf("ab: target percentage must be in (0, 1], got %v", exp.TargetPercentage)
	}

	totalWeight := 0.0
	ids := make(map[string]bool, len(exp.Variants))
	for _, variant := range exp.Variants {
		if variant.ID == "" {
			return errors.New("ab: variant ID is required")
		}
		if ids[variant.ID] {
			return fmt.Errorf("ab: duplicate variant ID %q", variant.ID)
		}
		ids[variant.ID] = true
		if !(variant.Weight >= 0) {
			return fmt.Errorf("ab: variant %q has negative weight %v", variant.ID, variant.Weight)
		}
		totalWeight += variant.Weight
	}
	if math.Abs(totalWeight-1.0) > 0.001 {
		return fmt.Errorf("variant weights must sum to 1.0, got %f", totalWeight)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if exp.ID == "" {
		exp.ID = generateID()
	}
	if _, ok := e.experiments[exp.ID]; ok {
		return fmt.Errorf("ab: experiment %s already exists", exp.ID)
	}

	now := time.Now()
	exp.CreatedAt = now
	exp.UpdatedAt = now

	if e.store != nil {
		if err := e.store.SaveExperiment(cloneExperiment(exp)); err != nil {
			return fmt.Errorf("failed to save experiment to store: %w", err)
		}
	}
	e.experiments[exp.ID] = exp

	return nil
}

// SetStatus changes the status of an experiment. Only active experiments
// assign users.
func (e *ExperimentEngine) SetStatus(experimentID string, status ExperimentStatus) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exp, ok := e.experiments[experimentID]
	if !ok {
		return fmt.Errorf("experiment %s not found", experimentID)
	}
	exp.Status = status
	exp.UpdatedAt = time.Now()
	return nil
}

// AssignVariant returns the user's assignment, creating it on first call.
// It returns nil, nil when the user is outside the target percentage or does
// not match the filters. Each assigned user counts once as an impression.
func (e *ExperimentEngine) AssignVariant(ctx context.Context, userID, experimentID string, userAttributes map[string]string) (*Assignment, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	exp, exists := e.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	if exp.Status != StatusActive {
		return nil, fmt.Errorf("experiment %s is not active", experimentID)
	}

	key := assignmentKey{userID, experimentID}
	if assignment, exists := e.assignments[key]; exists {
		return cloneAssignment(assignment), nil
	}

	if !e.shouldIncludeUser(userID, exp, userAttributes) {
		return nil, nil
	}

	i := e.selectVariant(userID, experimentID, exp.Variants)
	assignment := &Assignment{
		UserID:       userID,
		ExperimentID: experimentID,
		VariantID:    exp.Variants[i].ID,
		AssignedAt:   time.Now(),
		Attributes:   userAttributes,
	}

	if e.store != nil {
		if err := e.store.SaveAssignment(cloneAssignment(assignment)); err != nil {
			return nil, fmt.Errorf("failed to save assignment to store: %w", err)
		}
	}

	e.assignments[key] = assignment
	exp.Variants[i].Impressions++

	return cloneAssignment(assignment), nil
}

// shouldIncludeUser determines if a user should be included in the experiment
func (e *ExperimentEngine) shouldIncludeUser(userID string, exp *Experiment, userAttributes map[string]string) bool {
	if !rolloutFraction(userID+exp.ID, exp.TargetPercentage) {
		return false
	}

	for key, expectedValue := range exp.Filters {
		if actualValue, exists := userAttributes[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// selectVariant returns the index of the variant chosen by a stable hash of
// the user and experiment, in proportion to the variant weights.
func (e *ExperimentEngine) selectVariant(userID, experimentID string, variants []Variant) int {
	// u is in [0, 1).
	u := float64(Hash(userID+experimentID, 1000000)) / 1000000.0

	cumulative := 0.0
	last := 0
	for i, v := range variants {
		if v.Weight <= 0 {
			continue
		}
		cumulative += v.Weight
		last = i
		if u < cumulative {
			return i
		}
	}

	// The weights may sum to slightly less than 1.
	return last
}

// RecordConversion counts the first conversion of an assigned user. The
// event's VariantID, if set, must match the user's assignment. Every event
// is passed to the store.
func (e *ExperimentEngine) RecordConversion(ctx context.Context, event *ConversionEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exp, exists := e.experiments[event.ExperimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", event.ExperimentID)
	}

	key := assignmentKey{event.UserID, event.ExperimentID}
	assignment, ok := e.assignments[key]
	if !ok {
		return fmt.Errorf("ab: user %s is not assigned to experiment %s", event.UserID, event.ExperimentID)
	}
	if event.VariantID != "" && event.VariantID != assignment.VariantID {
		return fmt.Errorf("ab: user %s is assigned to variant %s, not %s", event.UserID, assignment.VariantID, event.VariantID)
	}

	if e.store != nil {
		if err := e.store.SaveConversion(event); err != nil {
			return fmt.Errorf("failed to save conversion event to store: %w", err)
		}
	}

	if e.converted[key] {
		return nil
	}
	e.converted[key] = true
	for i := range exp.Variants {
		if exp.Variants[i].ID == assignment.VariantID {
			exp.Variants[i].Conversions++
			break
		}
	}

	return nil
}

// GetExperimentResults reports each variant's conversion rate with a Wilson
// confidence interval, and a two-sided pooled z-test of each variant against
// the first (control) variant at the experiment's confidence level.
func (e *ExperimentEngine) GetExperimentResults(experimentID string) (*ExperimentResults, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	exp, exists := e.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	confidence := exp.ConfidenceLevel
	if !(confidence > 0 && confidence < 1) {
		confidence = defaultConfidenceLevel
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
			VariantID:          variant.ID,
			Impressions:        variant.Impressions,
			Conversions:        variant.Conversions,
			ConversionRate:     conversionRate,
			ConfidenceInterval: e.calculateConfidenceInterval(variant.Conversions, variant.Impressions, confidence),
		}
	}

	if len(results.Variants) >= 2 {
		control := results.Variants[0]
		for i := 1; i < len(results.Variants); i++ {
			pValue, _ := e.calculateZTest(control, results.Variants[i])
			results.Variants[i].PValue = pValue
			results.Variants[i].IsSignificant = pValue < 1.0-confidence
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

// calculateConfidenceInterval returns the Wilson interval for the conversion
// rate at the confidence level.
func (e *ExperimentEngine) calculateConfidenceInterval(conversions, impressions int64, confidenceLevel float64) ConfidenceInterval {
	return wilson(conversions, impressions, zScore(confidenceLevel))
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

	pooledP := (float64(control.Conversions) + float64(treatment.Conversions)) / (n1 + n2)
	se := math.Sqrt(pooledP * (1 - pooledP) * (1/n1 + 1/n2))
	if se == 0 {
		return 1.0, fmt.Errorf("standard error is zero")
	}

	z := (p2 - p1) / se

	// Two-tailed p-value.
	return math.Erfc(math.Abs(z) / math.Sqrt2), nil
}

func cloneExperiment(exp *Experiment) *Experiment {
	c := *exp
	c.Variants = append([]Variant(nil), exp.Variants...)
	return &c
}

func cloneAssignment(a *Assignment) *Assignment {
	c := *a
	return &c
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

// InMemoryExperimentStore is an ExperimentStore for tests and development.
// It is safe for concurrent use.
type InMemoryExperimentStore struct {
	mu          sync.RWMutex
	experiments map[string]*Experiment
	assignments map[assignmentKey]*Assignment
	conversions map[string][]*ConversionEvent // experimentID -> events
}

func NewInMemoryExperimentStore() *InMemoryExperimentStore {
	return &InMemoryExperimentStore{
		experiments: make(map[string]*Experiment),
		assignments: make(map[assignmentKey]*Assignment),
		conversions: make(map[string][]*ConversionEvent),
	}
}

func (s *InMemoryExperimentStore) SaveExperiment(exp *Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.experiments[exp.ID] = exp
	return nil
}

func (s *InMemoryExperimentStore) GetExperiment(id string) (*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.experiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment %s not found", id)
	}
	return exp, nil
}

func (s *InMemoryExperimentStore) ListExperiments() ([]*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exps := make([]*Experiment, 0, len(s.experiments))
	for _, exp := range s.experiments {
		exps = append(exps, exp)
	}
	return exps, nil
}

func (s *InMemoryExperimentStore) SaveAssignment(assignment *Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assignments[assignmentKey{assignment.UserID, assignment.ExperimentID}] = assignment
	return nil
}

func (s *InMemoryExperimentStore) GetAssignment(userID, experimentID string) (*Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assignment, ok := s.assignments[assignmentKey{userID, experimentID}]
	if !ok {
		return nil, fmt.Errorf("assignment not found")
	}
	return assignment, nil
}

func (s *InMemoryExperimentStore) ListAssignments(experimentID string) ([]*Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Assignment
	for _, a := range s.assignments {
		if a.ExperimentID == experimentID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (s *InMemoryExperimentStore) SaveConversion(event *ConversionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversions[event.ExperimentID] = append(s.conversions[event.ExperimentID], event)
	return nil
}

func (s *InMemoryExperimentStore) ListConversions(experimentID string) ([]*ConversionEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]*ConversionEvent(nil), s.conversions[experimentID]...), nil
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
