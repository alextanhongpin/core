package ab

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
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

// InMemoryBanditStorage is a BanditStorage for tests and development. It is
// safe for concurrent use.
type InMemoryBanditStorage struct {
	mu          sync.RWMutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.experiments[exp.ID] = exp
	return nil
}
func (s *InMemoryBanditStorage) SaveArm(expID string, arm *BanditArm) error {
	// No-op for in-memory, arms are part of experiment
	return nil
}
func (s *InMemoryBanditStorage) SaveResult(result BanditResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[result.ExperimentID] = append(s.results[result.ExperimentID], result)
	return nil
}

// ListResults returns a copy of the oldest limit results, or all of them
// when limit <= 0.
func (s *InMemoryBanditStorage) ListResults(expID string, limit int) ([]BanditResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := s.results[expID]
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return append([]BanditResult(nil), results...), nil
}

// BanditMetrics tracks operational metrics for bandit engine
type BanditMetrics struct {
	Pulls   int64
	Rewards int64
	Errors  int64
	Regret  []float64
}

// BanditEngine manages multi-armed bandit experiments. It is safe for
// concurrent use. Arms returned by SelectArm are snapshots.
type BanditEngine struct {
	experiments map[string]*BanditExperiment
	rng         *rand.Rand
	storage     BanditStorage
	metrics     BanditMetrics
	mu          sync.Mutex
}

// NewBanditEngine creates a new bandit engine with in-memory storage.
func NewBanditEngine() *BanditEngine {
	return NewBanditEngineWithStorage(nil, nil)
}

// NewBanditEngineWithStorage creates a new bandit engine with custom storage
// and initial metrics. Nil arguments use in-memory storage and zero metrics.
func NewBanditEngineWithStorage(storage BanditStorage, metrics *BanditMetrics) *BanditEngine {
	if storage == nil {
		storage = NewInMemoryBanditStorage()
	}
	if metrics == nil {
		metrics = &BanditMetrics{}
	}
	return &BanditEngine{
		experiments: make(map[string]*BanditExperiment),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		storage:     storage,
		metrics:     *metrics,
	}
}

// CreateBanditExperiment validates and registers exp. The engine takes
// ownership of exp: callers must not modify it afterwards. Arms need unique,
// non-empty IDs and Epsilon must be in [0, 1].
func (b *BanditEngine) CreateBanditExperiment(exp *BanditExperiment) error {
	if exp == nil {
		return errors.New("ab: experiment is nil")
	}
	if len(exp.Arms) == 0 {
		return errors.New("ab: bandit experiment needs at least one arm")
	}
	ids := make(map[string]bool, len(exp.Arms))
	for _, arm := range exp.Arms {
		if arm.ID == "" {
			return errors.New("ab: arm ID is required")
		}
		if ids[arm.ID] {
			return fmt.Errorf("ab: duplicate arm ID %q", arm.ID)
		}
		ids[arm.ID] = true
		if arm.Alpha < 0 || arm.Beta < 0 {
			return fmt.Errorf("ab: arm %q has a negative prior", arm.ID)
		}
	}
	if !(exp.Epsilon >= 0 && exp.Epsilon <= 1) {
		return fmt.Errorf("ab: epsilon must be in [0, 1], got %v", exp.Epsilon)
	}
	switch exp.Algorithm {
	case EpsilonGreedy, UCB, ThompsonSampling, BayesianBandit:
	default:
		return fmt.Errorf("unsupported bandit algorithm: %s", exp.Algorithm)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if exp.ID == "" {
		exp.ID = generateID()
	}
	if _, ok := b.experiments[exp.ID]; ok {
		return fmt.Errorf("ab: experiment %s already exists", exp.ID)
	}

	now := time.Now()
	for i := range exp.Arms {
		arm := &exp.Arms[i]
		if arm.Alpha == 0 {
			arm.Alpha = 1 // Prior belief of 1 success
		}
		if arm.Beta == 0 {
			arm.Beta = 1 // Prior belief of 1 failure
		}
		arm.CreatedAt = now
		arm.UpdatedAt = now
	}

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
	exp.StartTime = now
	exp.CreatedAt = now
	exp.UpdatedAt = now

	if err := b.storage.SaveExperiment(exp); err != nil {
		return fmt.Errorf("failed to save experiment to storage: %w", err)
	}
	b.experiments[exp.ID] = exp
	return nil
}

// SelectArm selects an arm with the experiment's algorithm and counts a pull.
// Report its outcome once with RecordReward.
func (b *BanditEngine) SelectArm(experimentID, userID string, context map[string]any) (*BanditArm, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

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
	case ThompsonSampling, BayesianBandit:
		selectedArm = b.selectThompsonSampling(exp)
	default:
		return nil, fmt.Errorf("unsupported bandit algorithm: %s", exp.Algorithm)
	}

	now := time.Now()
	selectedArm.Pulls++
	exp.TotalPulls++
	selectedArm.UpdatedAt = now
	exp.UpdatedAt = now
	b.metrics.Pulls++
	if err := b.storage.SaveArm(experimentID, selectedArm); err != nil {
		b.metrics.Errors++
		return nil, fmt.Errorf("failed to save arm to storage: %w", err)
	}

	arm := *selectedArm
	return &arm, nil
}

// RecordReward records the outcome of one pull of an arm.
func (b *BanditEngine) RecordReward(experimentID, armID, userID string, reward float64, success bool, context map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	exp, exists := b.experiments[experimentID]
	if !exists {
		b.metrics.Errors++
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	var arm *BanditArm
	for i := range exp.Arms {
		if exp.Arms[i].ID == armID {
			arm = &exp.Arms[i]
			break
		}
	}
	if arm == nil {
		b.metrics.Errors++
		return fmt.Errorf("arm %s not found in experiment %s", armID, experimentID)
	}

	arm.TotalReward += reward
	if success {
		arm.Rewards++
		arm.Alpha++
		b.metrics.Rewards++
	} else {
		arm.Beta++
	}
	now := time.Now()
	arm.UpdatedAt = now
	exp.UpdatedAt = now

	result := BanditResult{
		ExperimentID: experimentID,
		ArmID:        armID,
		UserID:       userID,
		Reward:       reward,
		Success:      success,
		Context:      context,
		Timestamp:    now,
	}
	if err := b.storage.SaveResult(result); err != nil {
		b.metrics.Errors++
		return fmt.Errorf("failed to save result to storage: %w", err)
	}

	return nil
}

// selectEpsilonGreedy implements epsilon-greedy arm selection
func (b *BanditEngine) selectEpsilonGreedy(exp *BanditExperiment) *BanditArm {
	if b.rng.Float64() < exp.Epsilon {
		return &exp.Arms[b.rng.Intn(len(exp.Arms))]
	}

	bestArm := &exp.Arms[0]
	bestReward := b.getAverageReward(bestArm)
	for i := 1; i < len(exp.Arms); i++ {
		arm := &exp.Arms[i]
		if avg := b.getAverageReward(arm); avg > bestReward {
			bestReward = avg
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
		if ucb := b.calculateUCB(arm, exp.TotalPulls, exp.Confidence); ucb > bestUCB {
			bestUCB = ucb
			bestArm = arm
		}
	}

	return bestArm
}

// selectThompsonSampling picks the arm with the largest draw from its Beta
// posterior.
func (b *BanditEngine) selectThompsonSampling(exp *BanditExperiment) *BanditArm {
	bestArm := &exp.Arms[0]
	bestSample := b.sampleBeta(bestArm.Alpha, bestArm.Beta)
	for i := 1; i < len(exp.Arms); i++ {
		arm := &exp.Arms[i]
		if sample := b.sampleBeta(arm.Alpha, arm.Beta); sample > bestSample {
			bestSample = sample
			bestArm = arm
		}
	}

	return bestArm
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

	confidenceBound := confidence * math.Sqrt(math.Log(float64(totalPulls))/float64(arm.Pulls))
	return b.getAverageReward(arm) + confidenceBound
}

// sampleBeta draws from Beta(alpha, beta) as X/(X+Y) with X ~ Gamma(alpha)
// and Y ~ Gamma(beta).
func (b *BanditEngine) sampleBeta(alpha, beta float64) float64 {
	x := b.sampleGamma(alpha, 1.0)
	y := b.sampleGamma(beta, 1.0)
	if x+y == 0 {
		return alpha / (alpha + beta)
	}
	return x / (x + y)
}

// sampleGamma draws from Gamma(shape, scale) with the Marsaglia-Tsang
// method. Shapes below 1 use the boost Gamma(shape+1) * U^(1/shape).
func (b *BanditEngine) sampleGamma(shape, scale float64) float64 {
	if shape < 1 {
		u := b.rng.Float64()
		return b.sampleGamma(shape+1, scale) * math.Pow(u, 1/shape)
	}

	d := shape - 1.0/3
	c := 1 / math.Sqrt(9*d)
	for {
		x := b.rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := b.rng.Float64()
		if u < 1-0.0331*x*x*x*x {
			return d * v * scale
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v * scale
		}
	}
}

// GetBanditResults returns analysis of bandit experiment results, with arms
// sorted by conversion rate, highest first.
func (b *BanditEngine) GetBanditResults(experimentID string) (*BanditAnalysis, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.analyze(experimentID)
}

func (b *BanditEngine) analyze(experimentID string) (*BanditAnalysis, error) {
	exp, exists := b.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	analysis := &BanditAnalysis{
		ExperimentID: experimentID,
		TotalPulls:   exp.TotalPulls,
		Arms:         make([]BanditArmAnalysis, len(exp.Arms)),
	}

	bestRate := bestConversionRate(exp.Arms)
	for i := range exp.Arms {
		arm := &exp.Arms[i]
		lowerBound, upperBound := b.calculateBetaConfidenceInterval(arm.Alpha, arm.Beta, 0.95)

		analysis.Arms[i] = BanditArmAnalysis{
			ArmID:          arm.ID,
			Pulls:          arm.Pulls,
			Rewards:        arm.Rewards,
			ConversionRate: conversionRate(arm),
			AverageReward:  b.getAverageReward(arm),
			ConfidenceInterval: ConfidenceInterval{
				Lower: lowerBound,
				Upper: upperBound,
			},
			PosteriorMean: arm.Alpha / (arm.Alpha + arm.Beta),
			Regret:        (bestRate - conversionRate(arm)) * float64(arm.Pulls),
		}
		analysis.TotalRegret += analysis.Arms[i].Regret
	}

	sort.SliceStable(analysis.Arms, func(i, j int) bool {
		return analysis.Arms[i].ConversionRate > analysis.Arms[j].ConversionRate
	})

	return analysis, nil
}

// BanditAnalysis contains analysis results for a bandit experiment
type BanditAnalysis struct {
	ExperimentID string              `json:"experiment_id"`
	TotalPulls   int64               `json:"total_pulls"`
	TotalRegret  float64             `json:"total_regret"`
	Arms         []BanditArmAnalysis `json:"arms"`
}

// BanditArmAnalysis contains analysis for a single bandit arm. Regret is
// the empirical regret: (best observed rate - this arm's rate) * pulls.
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

// calculateBetaConfidenceInterval approximates the central credible interval
// of Beta(alpha, beta) with a normal distribution.
func (b *BanditEngine) calculateBetaConfidenceInterval(alpha, beta, confidence float64) (float64, float64) {
	mean := alpha / (alpha + beta)
	variance := (alpha * beta) / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
	margin := zScore(confidence) * math.Sqrt(variance)

	return math.Max(0, mean-margin), math.Min(1, mean+margin)
}

func conversionRate(arm *BanditArm) float64 {
	if arm.Pulls == 0 {
		return 0
	}
	return float64(arm.Rewards) / float64(arm.Pulls)
}

func bestConversionRate(arms []BanditArm) float64 {
	best := 0.0
	for i := range arms {
		best = max(best, conversionRate(&arms[i]))
	}
	return best
}

// StopExperiment stops a bandit experiment
func (b *BanditEngine) StopExperiment(experimentID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

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

// GetExperimentRecommendation suggests stopping once there are at least 100
// pulls and the best arm's 95% interval lies above the runner-up's.
func (b *BanditEngine) GetExperimentRecommendation(experimentID string) (*BanditRecommendation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	analysis, err := b.analyze(experimentID)
	if err != nil {
		return nil, err
	}

	recommendation := &BanditRecommendation{
		ExperimentID: experimentID,
		Decision:     "continue",
		Confidence:   "medium",
		Reasons:      make([]string, 0),
	}

	if analysis.TotalPulls < 100 {
		recommendation.Reasons = append(recommendation.Reasons, "Need more data (minimum 100 pulls)")
		return recommendation, nil
	}

	if len(analysis.Arms) >= 2 {
		best := analysis.Arms[0]
		second := analysis.Arms[1]

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
	Decision     string   `json:"decision"` // "continue" or "stop"
	WinnerArmID  string   `json:"winner_arm_id,omitempty"`
	Confidence   string   `json:"confidence"` // "medium" or "high"
	Reasons      []string `json:"reasons"`
}

// Experiments returns snapshots of all experiments.
func (b *BanditEngine) Experiments() []*BanditExperiment {
	b.mu.Lock()
	defer b.mu.Unlock()
	exps := make([]*BanditExperiment, 0, len(b.experiments))
	for _, exp := range b.experiments {
		c := *exp
		c.Arms = append([]BanditArm(nil), exp.Arms...)
		exps = append(exps, &c)
	}
	return exps
}

// Metrics returns a snapshot of the engine's operational counters.
func (b *BanditEngine) Metrics() BanditMetrics {
	b.mu.Lock()
	defer b.mu.Unlock()
	m := b.metrics
	m.Regret = append([]float64(nil), b.metrics.Regret...)
	return m
}

// DeleteExperiment removes an experiment and reports whether it existed.
func (b *BanditEngine) DeleteExperiment(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.experiments[id]; ok {
		delete(b.experiments, id)
		return true
	}
	return false
}
