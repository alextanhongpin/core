package ab

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// BanditAlgorithm represents different bandit algorithms
type BanditAlgorithm string

const (
	EpsilonGreedy    BanditAlgorithm = "epsilon_greedy"
	UCB              BanditAlgorithm = "ucb"      // Upper Confidence Bound
	ThompsonSampling BanditAlgorithm = "thompson" // Thompson Sampling
	BayesianBandit   BanditAlgorithm = "bayesian" // Bayesian Bandit
)

// BanditArm represents a single arm in a multi-armed bandit
type BanditArm struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Pulls       int64   `json:"pulls"`        // Number of times this arm was pulled
	Rewards     int64   `json:"rewards"`      // Number of successful outcomes
	TotalReward float64 `json:"total_reward"` // Sum of all rewards

	// Bayesian parameters (for Thompson Sampling and Bayesian bandits)
	Alpha float64 `json:"alpha"` // Success count + 1 (prior)
	Beta  float64 `json:"beta"`  // Failure count + 1 (prior)

	// Configuration
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// BanditExperiment represents a multi-armed bandit experiment
type BanditExperiment struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Algorithm   BanditAlgorithm `json:"algorithm"`
	Arms        []BanditArm     `json:"arms"`

	// Algorithm parameters
	Epsilon      float64 `json:"epsilon"`       // For epsilon-greedy
	Confidence   float64 `json:"confidence"`    // For UCB
	ExploreRatio float64 `json:"explore_ratio"` // General exploration parameter

	// Experiment state
	TotalPulls int64      `json:"total_pulls"`
	Status     string     `json:"status"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// BanditResult represents the result of pulling a bandit arm
type BanditResult struct {
	ExperimentID string         `json:"experiment_id"`
	ArmID        string         `json:"arm_id"`
	UserID       string         `json:"user_id"`
	Reward       float64        `json:"reward"`
	Success      bool           `json:"success"`
	Context      map[string]any `json:"context"`
	Timestamp    time.Time      `json:"timestamp"`
}

// BanditStorage defines interface for persisting bandit experiments, arms, and results
// In production, implement with Redis, SQL, etc.
type BanditStorage interface {
	SaveExperiment(exp *BanditExperiment) error
	SaveArm(expID string, arm *BanditArm) error
	SaveResult(result BanditResult) error
	ListResults(expID string, limit int) ([]BanditResult, error)
}

// InMemoryBanditStorage is a simple in-memory implementation
type InMemoryBanditStorage struct {
	experiments map[string]*BanditExperiment
	results     map[string][]BanditResult // expID -> results
}

func NewInMemoryBanditStorage() *InMemoryBanditStorage {
	return &InMemoryBanditStorage{
		experiments: make(map[string]*BanditExperiment),
		results:     make(map[string][]BanditResult),
	}
}

func (s *InMemoryBanditStorage) SaveExperiment(exp *BanditExperiment) error {
	s.experiments[exp.ID] = exp
	return nil
}
func (s *InMemoryBanditStorage) SaveArm(expID string, arm *BanditArm) error {
	// No-op for in-memory, arms are part of experiment
	return nil
}
func (s *InMemoryBanditStorage) SaveResult(result BanditResult) error {
	s.results[result.ExperimentID] = append(s.results[result.ExperimentID], result)
	return nil
}
func (s *InMemoryBanditStorage) ListResults(expID string, limit int) ([]BanditResult, error) {
	results := s.results[expID]
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// BanditMetrics tracks operational metrics for bandit engine
type BanditMetrics struct {
	Pulls   int64
	Rewards int64
	Errors  int64
	Regret  []float64
}

// BanditEngine manages multi-armed bandit experiments
type BanditEngine struct {
	experiments map[string]*BanditExperiment
	results     []BanditResult
	rng         *rand.Rand
	storage     BanditStorage
	metrics     BanditMetrics
}

// NewBanditEngine creates a new bandit engine
func NewBanditEngine() *BanditEngine {
	return &BanditEngine{
		experiments: make(map[string]*BanditExperiment),
		results:     make([]BanditResult, 0),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		storage:     NewInMemoryBanditStorage(),
		metrics:     BanditMetrics{},
	}
}

// NewBanditEngineWithStorage creates a new bandit engine with custom storage and metrics
func NewBanditEngineWithStorage(storage BanditStorage, metrics *BanditMetrics) *BanditEngine {
	if storage == nil {
		storage = NewInMemoryBanditStorage()
	}
	if metrics == nil {
		metrics = &BanditMetrics{}
	}
	return &BanditEngine{
		experiments: make(map[string]*BanditExperiment),
		results:     make([]BanditResult, 0),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		storage:     storage,
		metrics:     *metrics,
	}
}

// CreateBanditExperiment creates a new bandit experiment
func (b *BanditEngine) CreateBanditExperiment(exp *BanditExperiment) error {
	if exp.ID == "" {
		exp.ID = generateID()
	}

	// Initialize arms with priors
	for i := range exp.Arms {
		arm := &exp.Arms[i]
		if arm.Alpha == 0 {
			arm.Alpha = 1 // Prior belief of 1 success
		}
		if arm.Beta == 0 {
			arm.Beta = 1 // Prior belief of 1 failure
		}
		arm.CreatedAt = time.Now()
		arm.UpdatedAt = time.Now()
	}

	// Set default algorithm parameters
	if exp.Epsilon == 0 && exp.Algorithm == EpsilonGreedy {
		exp.Epsilon = 0.1 // 10% exploration
	}
	if exp.Confidence == 0 && exp.Algorithm == UCB {
		exp.Confidence = 2.0
	}
	if exp.ExploreRatio == 0 {
		exp.ExploreRatio = 0.1
	}

	exp.Status = "active"
	exp.StartTime = time.Now()
	exp.CreatedAt = time.Now()
	exp.UpdatedAt = time.Now()

	b.experiments[exp.ID] = exp
	_ = b.storage.SaveExperiment(exp)
	return nil
}

// SelectArm selects an arm based on the bandit algorithm
func (b *BanditEngine) SelectArm(experimentID, userID string, context map[string]any) (*BanditArm, error) {
	exp, exists := b.experiments[experimentID]
	if !exists {
		b.metrics.Errors++
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	if exp.Status != "active" {
		b.metrics.Errors++
		return nil, fmt.Errorf("experiment %s is not active", experimentID)
	}

	var selectedArm *BanditArm

	switch exp.Algorithm {
	case EpsilonGreedy:
		selectedArm = b.selectEpsilonGreedy(exp)
	case UCB:
		selectedArm = b.selectUCB(exp)
	case ThompsonSampling:
		selectedArm = b.selectThompsonSampling(exp)
	case BayesianBandit:
		selectedArm = b.selectBayesian(exp)
	default:
		return nil, fmt.Errorf("unsupported bandit algorithm: %s", exp.Algorithm)
	}

	if selectedArm == nil {
		return nil, fmt.Errorf("failed to select arm")
	}

	selectedArm.Pulls++
	exp.TotalPulls++
	selectedArm.UpdatedAt = time.Now()
	exp.UpdatedAt = time.Now()
	_ = b.storage.SaveArm(experimentID, selectedArm)
	b.metrics.Pulls++

	return selectedArm, nil
}

// RecordReward records the reward for a bandit arm pull
func (b *BanditEngine) RecordReward(experimentID, armID, userID string, reward float64, success bool, context map[string]any) error {
	exp, exists := b.experiments[experimentID]
	if !exists {
		b.metrics.Errors++
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	// Find the arm
	var arm *BanditArm
	for i := range exp.Arms {
		if exp.Arms[i].ID == armID {
			arm = &exp.Arms[i]
			break
		}
	}

	if arm == nil {
		return fmt.Errorf("arm %s not found in experiment %s", armID, experimentID)
	}

	// Update arm statistics
	arm.TotalReward += reward
	if success {
		arm.Rewards++
		arm.Alpha++ // Increment success count for Bayesian
		b.metrics.Rewards++
	} else {
		arm.Beta++ // Increment failure count for Bayesian
	}
	arm.UpdatedAt = time.Now()
	exp.UpdatedAt = time.Now()

	// Record the result
	result := BanditResult{
		ExperimentID: experimentID,
		ArmID:        armID,
		UserID:       userID,
		Reward:       reward,
		Success:      success,
		Context:      context,
		Timestamp:    time.Now(),
	}
	b.results = append(b.results, result)
	_ = b.storage.SaveResult(result)

	return nil
}

// selectEpsilonGreedy implements epsilon-greedy arm selection
func (b *BanditEngine) selectEpsilonGreedy(exp *BanditExperiment) *BanditArm {
	// Explore with probability epsilon
	if b.rng.Float64() < exp.Epsilon {
		// Random exploration
		return &exp.Arms[b.rng.Intn(len(exp.Arms))]
	}

	// Exploit: select arm with highest average reward
	bestArm := &exp.Arms[0]
	bestReward := b.getAverageReward(bestArm)

	for i := 1; i < len(exp.Arms); i++ {
		arm := &exp.Arms[i]
		avgReward := b.getAverageReward(arm)
		if avgReward > bestReward {
			bestReward = avgReward
			bestArm = arm
		}
	}

	return bestArm
}

// selectUCB implements Upper Confidence Bound arm selection
func (b *BanditEngine) selectUCB(exp *BanditExperiment) *BanditArm {
	bestArm := &exp.Arms[0]
	bestUCB := b.calculateUCB(bestArm, exp.TotalPulls, exp.Confidence)

	for i := 1; i < len(exp.Arms); i++ {
		arm := &exp.Arms[i]
		ucb := b.calculateUCB(arm, exp.TotalPulls, exp.Confidence)
		if ucb > bestUCB {
			bestUCB = ucb
			bestArm = arm
		}
	}

	return bestArm
}

// selectThompsonSampling implements Thompson Sampling arm selection
func (b *BanditEngine) selectThompsonSampling(exp *BanditExperiment) *BanditArm {
	bestArm := &exp.Arms[0]
	bestSample := b.sampleBeta(bestArm.Alpha, bestArm.Beta)

	for i := 1; i < len(exp.Arms); i++ {
		arm := &exp.Arms[i]
		sample := b.sampleBeta(arm.Alpha, arm.Beta)
		if sample > bestSample {
			bestSample = sample
			bestArm = arm
		}
	}

	return bestArm
}

// selectBayesian implements Bayesian bandit arm selection
func (b *BanditEngine) selectBayesian(exp *BanditExperiment) *BanditArm {
	// Similar to Thompson Sampling but with additional considerations
	return b.selectThompsonSampling(exp)
}

// getAverageReward calculates the average reward for an arm
func (b *BanditEngine) getAverageReward(arm *BanditArm) float64 {
	if arm.Pulls == 0 {
		return 0.0
	}
	return arm.TotalReward / float64(arm.Pulls)
}

// calculateUCB calculates the Upper Confidence Bound for an arm
func (b *BanditEngine) calculateUCB(arm *BanditArm, totalPulls int64, confidence float64) float64 {
	if arm.Pulls == 0 {
		return math.Inf(1) // Infinite UCB for unplayed arms
	}

	avgReward := b.getAverageReward(arm)
	confidenceBound := confidence * math.Sqrt(math.Log(float64(totalPulls))/float64(arm.Pulls))

	return avgReward + confidenceBound
}

// sampleBeta samples from a Beta distribution
func (b *BanditEngine) sampleBeta(alpha, beta float64) float64 {
	// Simple approximation using ratio of Gamma samples
	// For production use, consider a proper Beta distribution sampler
	x := b.sampleGamma(alpha, 1.0)
	y := b.sampleGamma(beta, 1.0)
	return x / (x + y)
}

// sampleGamma samples from a Gamma distribution (simplified implementation)
func (b *BanditEngine) sampleGamma(shape, scale float64) float64 {
	// Simplified Gamma sampling using normal approximation for large shape
	if shape > 10 {
		// Normal approximation
		mean := shape * scale
		stddev := math.Sqrt(shape) * scale
		return math.Max(0, b.rng.NormFloat64()*stddev+mean)
	}

	// For small shape, use exponential approximation
	return -math.Log(b.rng.Float64()) * scale
}

// GetBanditResults returns analysis of bandit experiment results
func (b *BanditEngine) GetBanditResults(experimentID string) (*BanditAnalysis, error) {
	exp, exists := b.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	analysis := &BanditAnalysis{
		ExperimentID: experimentID,
		TotalPulls:   exp.TotalPulls,
		Arms:         make([]BanditArmAnalysis, len(exp.Arms)),
	}

	for i, arm := range exp.Arms {
		conversionRate := 0.0
		if arm.Pulls > 0 {
			conversionRate = float64(arm.Rewards) / float64(arm.Pulls)
		}

		avgReward := b.getAverageReward(&arm)

		// Calculate confidence intervals
		lowerBound, upperBound := b.calculateBetaConfidenceInterval(arm.Alpha, arm.Beta, 0.95)

		analysis.Arms[i] = BanditArmAnalysis{
			ArmID:          arm.ID,
			Pulls:          arm.Pulls,
			Rewards:        arm.Rewards,
			ConversionRate: conversionRate,
			AverageReward:  avgReward,
			ConfidenceInterval: ConfidenceInterval{
				Lower: lowerBound,
				Upper: upperBound,
			},
			PosteriorMean: arm.Alpha / (arm.Alpha + arm.Beta),
			Regret:        b.calculateRegret(&arm, analysis),
		}
	}

	// Sort arms by conversion rate
	sort.Slice(analysis.Arms, func(i, j int) bool {
		return analysis.Arms[i].ConversionRate > analysis.Arms[j].ConversionRate
	})

	// Calculate total regret
	analysis.TotalRegret = b.calculateTotalRegret(exp)

	return analysis, nil
}

// BanditAnalysis contains analysis results for a bandit experiment
type BanditAnalysis struct {
	ExperimentID string              `json:"experiment_id"`
	TotalPulls   int64               `json:"total_pulls"`
	TotalRegret  float64             `json:"total_regret"`
	Arms         []BanditArmAnalysis `json:"arms"`
}

// BanditArmAnalysis contains analysis for a single bandit arm
type BanditArmAnalysis struct {
	ArmID              string             `json:"arm_id"`
	Pulls              int64              `json:"pulls"`
	Rewards            int64              `json:"rewards"`
	ConversionRate     float64            `json:"conversion_rate"`
	AverageReward      float64            `json:"average_reward"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	PosteriorMean      float64            `json:"posterior_mean"`
	Regret             float64            `json:"regret"`
}

// calculateBetaConfidenceInterval calculates confidence interval for Beta distribution
func (b *BanditEngine) calculateBetaConfidenceInterval(alpha, beta, confidence float64) (float64, float64) {
	// Simplified calculation using normal approximation
	mean := alpha / (alpha + beta)
	variance := (alpha * beta) / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
	stddev := math.Sqrt(variance)

	// Z-score for confidence level
	z := 1.96 // For 95% confidence
	switch confidence {
	case 0.99:
		z = 2.576
	case 0.90:
		z = 1.645
	}

	margin := z * stddev

	return math.Max(0, mean-margin), math.Min(1, mean+margin)
}

// calculateRegret calculates regret for an arm compared to the best arm
func (b *BanditEngine) calculateRegret(arm *BanditArm, analysis *BanditAnalysis) float64 {
	if len(analysis.Arms) == 0 {
		return 0.0
	}

	// Find the best arm's conversion rate
	bestRate := 0.0
	for _, armAnalysis := range analysis.Arms {
		if armAnalysis.ConversionRate > bestRate {
			bestRate = armAnalysis.ConversionRate
		}
	}

	currentRate := 0.0
	if arm.Pulls > 0 {
		currentRate = float64(arm.Rewards) / float64(arm.Pulls)
	}

	return (bestRate - currentRate) * float64(arm.Pulls)
}

// calculateTotalRegret calculates total regret for the experiment
func (b *BanditEngine) calculateTotalRegret(exp *BanditExperiment) float64 {
	// Find the best arm's true conversion rate
	bestRate := 0.0
	for _, arm := range exp.Arms {
		rate := 0.0
		if arm.Pulls > 0 {
			rate = float64(arm.Rewards) / float64(arm.Pulls)
		}
		if rate > bestRate {
			bestRate = rate
		}
	}

	totalRegret := 0.0
	for _, arm := range exp.Arms {
		currentRate := 0.0
		if arm.Pulls > 0 {
			currentRate = float64(arm.Rewards) / float64(arm.Pulls)
		}
		totalRegret += (bestRate - currentRate) * float64(arm.Pulls)
	}

	return totalRegret
}

// StopExperiment stops a bandit experiment
func (b *BanditEngine) StopExperiment(experimentID string) error {
	exp, exists := b.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	exp.Status = "stopped"
	now := time.Now()
	exp.EndTime = &now
	exp.UpdatedAt = now

	return nil
}

// GetExperimentRecommendation provides recommendations for experiment decisions
func (b *BanditEngine) GetExperimentRecommendation(experimentID string) (*BanditRecommendation, error) {
	analysis, err := b.GetBanditResults(experimentID)
	if err != nil {
		return nil, err
	}

	exp := b.experiments[experimentID]

	recommendation := &BanditRecommendation{
		ExperimentID: experimentID,
		Decision:     "continue",
		Confidence:   "medium",
		Reasons:      make([]string, 0),
	}

	// Check if we have enough data
	if exp.TotalPulls < 100 {
		recommendation.Reasons = append(recommendation.Reasons, "Need more data (minimum 100 pulls)")
		return recommendation, nil
	}

	// Check if there's a clear winner
	if len(analysis.Arms) >= 2 {
		best := analysis.Arms[0]
		second := analysis.Arms[1]

		// If confidence intervals don't overlap and we have enough data
		if best.ConfidenceInterval.Lower > second.ConfidenceInterval.Upper && best.Pulls > 50 {
			recommendation.Decision = "stop"
			recommendation.WinnerArmID = best.ArmID
			recommendation.Confidence = "high"
			recommendation.Reasons = append(recommendation.Reasons,
				fmt.Sprintf("Clear winner: %s with %.2f%% conversion rate",
					best.ArmID, best.ConversionRate*100))
		}
	}

	return recommendation, nil
}

// BanditRecommendation provides recommendations for bandit experiment decisions
type BanditRecommendation struct {
	ExperimentID string   `json:"experiment_id"`
	Decision     string   `json:"decision"` // "continue", "stop", "need_more_data"
	WinnerArmID  string   `json:"winner_arm_id,omitempty"`
	Confidence   string   `json:"confidence"` // "low", "medium", "high"
	Reasons      []string `json:"reasons"`
}
