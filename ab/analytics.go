package ab

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricConversion   MetricType = "conversion"
	MetricRevenue      MetricType = "revenue"
	MetricEngagement   MetricType = "engagement"
	MetricRetention    MetricType = "retention"
	MetricClickThrough MetricType = "click_through"
	MetricCustom       MetricType = "custom"
)

// Metric represents a metric definition
type Metric struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Description string                 `json:"description"`
	Unit        string                 `json:"unit"`       // e.g., "percent", "dollars", "count"
	IsPrimary   bool                   `json:"is_primary"` // Primary metric for the experiment
	Properties  map[string]interface{} `json:"properties"`
	CreatedAt   time.Time              `json:"created_at"`
}

// MetricValue represents a measured value for a metric
type MetricValue struct {
	MetricID     string                 `json:"metric_id"`
	ExperimentID string                 `json:"experiment_id"`
	VariantID    string                 `json:"variant_id"`
	UserID       string                 `json:"user_id"`
	Value        float64                `json:"value"`
	Properties   map[string]interface{} `json:"properties"`
	Timestamp    time.Time              `json:"timestamp"`
}

// SegmentCriteria defines criteria for user segmentation
type SegmentCriteria struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"` // "equals", "contains", "greater_than", etc.
	Value     interface{} `json:"value"`
}

// Segment represents a user segment for analysis
type Segment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Criteria    []SegmentCriteria `json:"criteria"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ExperimentAnalytics provides comprehensive analytics for experiments
type ExperimentAnalytics struct {
	ExperimentID     string                     `json:"experiment_id"`
	TimeRange        TimeRange                  `json:"time_range"`
	OverallMetrics   []MetricSummary            `json:"overall_metrics"`
	VariantMetrics   map[string][]MetricSummary `json:"variant_metrics"`  // variant_id -> metrics
	SegmentAnalysis  map[string]SegmentAnalysis `json:"segment_analysis"` // segment_id -> analysis
	TimeSeriesData   []TimeSeriesPoint          `json:"time_series_data"`
	StatisticalTests []StatisticalTest          `json:"statistical_tests"`
	GeneratedAt      time.Time                  `json:"generated_at"`
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricSummary provides summary statistics for a metric
type MetricSummary struct {
	MetricID           string             `json:"metric_id"`
	MetricName         string             `json:"metric_name"`
	Count              int64              `json:"count"`
	Sum                float64            `json:"sum"`
	Mean               float64            `json:"mean"`
	Median             float64            `json:"median"`
	StandardDev        float64            `json:"standard_deviation"`
	Percentiles        map[int]float64    `json:"percentiles"` // 25th, 75th, 90th, 95th, 99th
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
}

// SegmentAnalysis provides analysis for a specific segment
type SegmentAnalysis struct {
	SegmentID      string                     `json:"segment_id"`
	SegmentName    string                     `json:"segment_name"`
	UserCount      int64                      `json:"user_count"`
	VariantMetrics map[string][]MetricSummary `json:"variant_metrics"`
	Insights       []string                   `json:"insights"`
}

// TimeSeriesPoint represents a point in time series data
type TimeSeriesPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	VariantID string                 `json:"variant_id"`
	MetricID  string                 `json:"metric_id"`
	Value     float64                `json:"value"`
	Count     int64                  `json:"count"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// StatisticalTest represents results of a statistical test
type StatisticalTest struct {
	TestType         string  `json:"test_type"` // "t_test", "chi_square", "mann_whitney"
	MetricID         string  `json:"metric_id"`
	ControlVariant   string  `json:"control_variant"`
	TreatmentVariant string  `json:"treatment_variant"`
	PValue           float64 `json:"p_value"`
	TestStatistic    float64 `json:"test_statistic"`
	DegreesOfFreedom int     `json:"degrees_of_freedom,omitempty"`
	EffectSize       float64 `json:"effect_size"`
	IsSignificant    bool    `json:"is_significant"`
	ConfidenceLevel  float64 `json:"confidence_level"`
}

// AnalyticsStorage defines the interface for persistent analytics storage
// (e.g., for metric values, segments, etc.)
type AnalyticsStorage interface {
	SaveMetricValue(ctx context.Context, value MetricValue) error
	GetMetricValues(ctx context.Context, experimentID string, timeRange TimeRange) ([]MetricValue, error)
	SaveSegment(ctx context.Context, segment *Segment) error
	GetSegments(ctx context.Context) ([]*Segment, error)
}

// InMemoryAnalyticsStorage is an in-memory implementation of AnalyticsStorage
// (for testing and development)
type InMemoryAnalyticsStorage struct {
	metricValues []MetricValue
	segments     map[string]*Segment
}

func NewInMemoryAnalyticsStorage() *InMemoryAnalyticsStorage {
	return &InMemoryAnalyticsStorage{
		metricValues: make([]MetricValue, 0),
		segments:     make(map[string]*Segment),
	}
}

func (s *InMemoryAnalyticsStorage) SaveMetricValue(ctx context.Context, value MetricValue) error {
	s.metricValues = append(s.metricValues, value)
	return nil
}

func (s *InMemoryAnalyticsStorage) GetMetricValues(ctx context.Context, experimentID string, timeRange TimeRange) ([]MetricValue, error) {
	var values []MetricValue
	for _, v := range s.metricValues {
		if v.ExperimentID == experimentID &&
			v.Timestamp.After(timeRange.Start) &&
			v.Timestamp.Before(timeRange.End) {
			values = append(values, v)
		}
	}
	return values, nil
}

func (s *InMemoryAnalyticsStorage) SaveSegment(ctx context.Context, segment *Segment) error {
	s.segments[segment.ID] = segment
	return nil
}

func (s *InMemoryAnalyticsStorage) GetSegments(ctx context.Context) ([]*Segment, error) {
	var segs []*Segment
	for _, seg := range s.segments {
		segs = append(segs, seg)
	}
	return segs, nil
}

// AnalyticsMetrics holds operational metrics for observability
// (e.g., counts, error rates, latency, etc.)
type AnalyticsMetrics struct {
	MetricValueCount int64
	SegmentCount     int64
	LastError        error
	LastOperation    string
}

// AnalyticsEngine provides comprehensive analytics capabilities
// Now supports pluggable storage and metrics
type AnalyticsEngine struct {
	metrics     map[string]*Metric
	storage     AnalyticsStorage
	metricsObs  *AnalyticsMetrics
	segments    map[string]*Segment
	assignments map[string]*Assignment // From experiment engine
}

// NewAnalyticsEngine creates a new analytics engine with storage and metrics
func NewAnalyticsEngineWithStorage(storage AnalyticsStorage, metricsObs *AnalyticsMetrics) *AnalyticsEngine {
	if storage == nil {
		storage = NewInMemoryAnalyticsStorage()
	}
	if metricsObs == nil {
		metricsObs = &AnalyticsMetrics{}
	}
	return &AnalyticsEngine{
		metrics:     make(map[string]*Metric),
		storage:     storage,
		metricsObs:  metricsObs,
		segments:    make(map[string]*Segment),
		assignments: make(map[string]*Assignment),
	}
}

// NewAnalyticsEngine creates a new analytics engine (default in-memory)
func NewAnalyticsEngine() *AnalyticsEngine {
	return NewAnalyticsEngineWithStorage(nil, nil)
}

// CreateMetric creates a new metric definition
func (a *AnalyticsEngine) CreateMetric(metric *Metric) error {
	if metric.ID == "" {
		metric.ID = generateID()
	}
	metric.CreatedAt = time.Now()
	a.metrics[metric.ID] = metric
	return nil
}

// RecordMetricValue records a value for a metric (now uses storage and metrics)
func (a *AnalyticsEngine) RecordMetricValue(ctx context.Context, value MetricValue) error {
	value.Timestamp = time.Now()
	err := a.storage.SaveMetricValue(ctx, value)
	if err == nil {
		a.metricsObs.MetricValueCount++
		a.metricsObs.LastOperation = "RecordMetricValue"
	} else {
		a.metricsObs.LastError = err
	}
	return err
}

// CreateSegment creates a new user segment (now uses storage and metrics)
func (a *AnalyticsEngine) CreateSegment(segment *Segment) error {
	if segment.ID == "" {
		segment.ID = generateID()
	}
	segment.CreatedAt = time.Now()
	err := a.storage.SaveSegment(context.Background(), segment)
	if err == nil {
		a.metricsObs.SegmentCount++
		a.metricsObs.LastOperation = "CreateSegment"
	} else {
		a.metricsObs.LastError = err
	}
	a.segments[segment.ID] = segment // keep in-memory for fast access
	return err
}

// GenerateAnalytics generates comprehensive analytics for an experiment (now uses storage)
func (a *AnalyticsEngine) GenerateAnalytics(ctx context.Context, experimentID string, timeRange TimeRange) (*ExperimentAnalytics, error) {
	analytics := &ExperimentAnalytics{
		ExperimentID:    experimentID,
		TimeRange:       timeRange,
		VariantMetrics:  make(map[string][]MetricSummary),
		SegmentAnalysis: make(map[string]SegmentAnalysis),
		GeneratedAt:     time.Now(),
	}

	// Get all metric values for this experiment in the time range
	experimentValues, err := a.storage.GetMetricValues(ctx, experimentID, timeRange)
	if err != nil {
		a.metricsObs.LastError = err
		return nil, err
	}

	// Calculate overall metrics
	analytics.OverallMetrics = a.calculateOverallMetrics(experimentValues)

	// Calculate variant-specific metrics
	for variantID := range a.getExperimentVariants(experimentID) {
		variantValues := a.filterByVariant(experimentValues, variantID)
		analytics.VariantMetrics[variantID] = a.calculateMetricSummaries(variantValues)
	}

	// Generate time series data
	analytics.TimeSeriesData = a.generateTimeSeriesData(experimentValues, timeRange)

	// Perform statistical tests
	analytics.StatisticalTests = a.performStatisticalTests(analytics.VariantMetrics)

	// Analyze segments
	segs, err := a.storage.GetSegments(ctx)
	if err != nil {
		a.metricsObs.LastError = err
		return nil, err
	}
	for _, segment := range segs {
		segmentAnalysis := a.analyzeSegment(experimentID, segment.ID, timeRange)
		analytics.SegmentAnalysis[segment.ID] = segmentAnalysis
	}

	return analytics, nil
}

// getExperimentVariants gets all variant IDs for an experiment
func (a *AnalyticsEngine) getExperimentVariants(experimentID string) map[string]bool {
	variants := make(map[string]bool)

	// Get variants from assignments
	for _, assignment := range a.assignments {
		if assignment.ExperimentID == experimentID {
			variants[assignment.VariantID] = true
		}
	}

	// Also get variants from metric values
	values, err := a.storage.GetMetricValues(context.Background(), experimentID, TimeRange{})
	if err == nil {
		for _, value := range values {
			variants[value.VariantID] = true
		}
	}

	return variants
}

// filterByVariant filters metric values by variant ID
func (a *AnalyticsEngine) filterByVariant(values []MetricValue, variantID string) []MetricValue {
	var filtered []MetricValue

	for _, value := range values {
		if value.VariantID == variantID {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

// calculateOverallMetrics calculates overall metrics across all variants
func (a *AnalyticsEngine) calculateOverallMetrics(values []MetricValue) []MetricSummary {
	metricGroups := make(map[string][]float64)

	// Group values by metric ID
	for _, value := range values {
		metricGroups[value.MetricID] = append(metricGroups[value.MetricID], value.Value)
	}

	var summaries []MetricSummary
	for metricID, metricValues := range metricGroups {
		summary := a.calculateMetricSummary(metricID, metricValues)
		summaries = append(summaries, summary)
	}

	return summaries
}

// calculateMetricSummaries calculates metric summaries for a set of values
func (a *AnalyticsEngine) calculateMetricSummaries(values []MetricValue) []MetricSummary {
	metricGroups := make(map[string][]float64)

	// Group values by metric ID
	for _, value := range values {
		metricGroups[value.MetricID] = append(metricGroups[value.MetricID], value.Value)
	}

	var summaries []MetricSummary
	for metricID, metricValues := range metricGroups {
		summary := a.calculateMetricSummary(metricID, metricValues)
		summaries = append(summaries, summary)
	}

	return summaries
}

// calculateMetricSummary calculates summary statistics for a metric
func (a *AnalyticsEngine) calculateMetricSummary(metricID string, values []float64) MetricSummary {
	if len(values) == 0 {
		return MetricSummary{MetricID: metricID}
	}

	// Sort values for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate basic statistics
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	standardDev := math.Sqrt(variance)

	// Calculate percentiles
	percentiles := make(map[int]float64)
	percentiles[25] = a.calculatePercentile(sorted, 0.25)
	percentiles[50] = a.calculatePercentile(sorted, 0.50) // median
	percentiles[75] = a.calculatePercentile(sorted, 0.75)
	percentiles[90] = a.calculatePercentile(sorted, 0.90)
	percentiles[95] = a.calculatePercentile(sorted, 0.95)
	percentiles[99] = a.calculatePercentile(sorted, 0.99)

	// Calculate confidence interval (95%)
	standardError := standardDev / math.Sqrt(float64(len(values)))
	margin := 1.96 * standardError // 95% confidence

	metricName := metricID
	if metric, exists := a.metrics[metricID]; exists {
		metricName = metric.Name
	}

	return MetricSummary{
		MetricID:    metricID,
		MetricName:  metricName,
		Count:       int64(len(values)),
		Sum:         sum,
		Mean:        mean,
		Median:      percentiles[50],
		StandardDev: standardDev,
		Percentiles: percentiles,
		ConfidenceInterval: ConfidenceInterval{
			Lower: mean - margin,
			Upper: mean + margin,
		},
	}
}

// calculatePercentile calculates the value at a given percentile
func (a *AnalyticsEngine) calculatePercentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := percentile * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// generateTimeSeriesData generates time series data for visualization
func (a *AnalyticsEngine) generateTimeSeriesData(values []MetricValue, timeRange TimeRange) []TimeSeriesPoint {
	// Group by time buckets (e.g., hourly or daily)
	bucketSize := a.calculateBucketSize(timeRange)
	buckets := make(map[time.Time]map[string]map[string][]float64) // time -> variant -> metric -> values

	for _, value := range values {
		bucket := value.Timestamp.Truncate(bucketSize)

		if buckets[bucket] == nil {
			buckets[bucket] = make(map[string]map[string][]float64)
		}
		if buckets[bucket][value.VariantID] == nil {
			buckets[bucket][value.VariantID] = make(map[string][]float64)
		}

		buckets[bucket][value.VariantID][value.MetricID] = append(
			buckets[bucket][value.VariantID][value.MetricID], value.Value)
	}

	var points []TimeSeriesPoint
	for timestamp, variants := range buckets {
		for variantID, metrics := range variants {
			for metricID, metricValues := range metrics {
				sum := 0.0
				for _, v := range metricValues {
					sum += v
				}

				points = append(points, TimeSeriesPoint{
					Timestamp: timestamp,
					VariantID: variantID,
					MetricID:  metricID,
					Value:     sum / float64(len(metricValues)),
					Count:     int64(len(metricValues)),
				})
			}
		}
	}

	// Sort by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points
}

// calculateBucketSize determines the appropriate time bucket size
func (a *AnalyticsEngine) calculateBucketSize(timeRange TimeRange) time.Duration {
	duration := timeRange.End.Sub(timeRange.Start)

	if duration <= 24*time.Hour {
		return time.Hour
	} else if duration <= 7*24*time.Hour {
		return 4 * time.Hour
	} else if duration <= 30*24*time.Hour {
		return 24 * time.Hour
	} else {
		return 7 * 24 * time.Hour
	}
}

// performStatisticalTests performs statistical tests between variants
func (a *AnalyticsEngine) performStatisticalTests(variantMetrics map[string][]MetricSummary) []StatisticalTest {
	var tests []StatisticalTest

	// Get variant IDs
	var variantIDs []string
	for variantID := range variantMetrics {
		variantIDs = append(variantIDs, variantID)
	}

	if len(variantIDs) < 2 {
		return tests
	}

	// Use first variant as control
	controlID := variantIDs[0]

	// Compare each treatment variant with control
	for i := 1; i < len(variantIDs); i++ {
		treatmentID := variantIDs[i]

		// Test each metric
		controlMetrics := variantMetrics[controlID]
		treatmentMetrics := variantMetrics[treatmentID]

		for _, controlMetric := range controlMetrics {
			for _, treatmentMetric := range treatmentMetrics {
				if controlMetric.MetricID == treatmentMetric.MetricID {
					test := a.performTTest(controlMetric, treatmentMetric, controlID, treatmentID)
					tests = append(tests, test)
				}
			}
		}
	}

	return tests
}

// performTTest performs a t-test between two metric summaries
func (a *AnalyticsEngine) performTTest(control, treatment MetricSummary, controlID, treatmentID string) StatisticalTest {
	// Simplified t-test calculation
	n1 := float64(control.Count)
	n2 := float64(treatment.Count)

	if n1 == 0 || n2 == 0 {
		return StatisticalTest{
			TestType:         "t_test",
			MetricID:         control.MetricID,
			ControlVariant:   controlID,
			TreatmentVariant: treatmentID,
			PValue:           1.0,
			IsSignificant:    false,
			ConfidenceLevel:  0.95,
		}
	}

	mean1 := control.Mean
	mean2 := treatment.Mean
	std1 := control.StandardDev
	std2 := treatment.StandardDev

	// Pooled standard error
	pooledSE := math.Sqrt((std1*std1)/n1 + (std2*std2)/n2)

	// T-statistic
	tStat := (mean2 - mean1) / pooledSE

	// Degrees of freedom (Welch's formula approximation)
	df := int((std1*std1/n1 + std2*std2/n2) * (std1*std1/n1 + std2*std2/n2) /
		((std1*std1/n1)*(std1*std1/n1)/(n1-1) + (std2*std2/n2)*(std2*std2/n2)/(n2-1)))

	// Simplified p-value calculation (would need proper t-distribution in production)
	pValue := 2 * (1 - math.Abs(tStat)/3.0) // Very simplified approximation
	if pValue < 0 {
		pValue = 0
	}
	if pValue > 1 {
		pValue = 1
	}

	// Effect size (Cohen's d)
	pooledStd := math.Sqrt(((n1-1)*std1*std1 + (n2-1)*std2*std2) / (n1 + n2 - 2))
	effectSize := (mean2 - mean1) / pooledStd

	return StatisticalTest{
		TestType:         "t_test",
		MetricID:         control.MetricID,
		ControlVariant:   controlID,
		TreatmentVariant: treatmentID,
		PValue:           pValue,
		TestStatistic:    tStat,
		DegreesOfFreedom: df,
		EffectSize:       effectSize,
		IsSignificant:    pValue < 0.05,
		ConfidenceLevel:  0.95,
	}
}

// analyzeSegment analyzes metrics for a specific user segment
func (a *AnalyticsEngine) analyzeSegment(experimentID, segmentID string, timeRange TimeRange) SegmentAnalysis {
	segment := a.segments[segmentID]
	if segment == nil {
		return SegmentAnalysis{SegmentID: segmentID}
	}

	// Get users in this segment
	segmentUsers := a.getUsersInSegment(segmentID)

	// Filter metric values for segment users
	segmentValues := make([]MetricValue, 0)
	values, err := a.storage.GetMetricValues(context.Background(), experimentID, timeRange)
	if err == nil {
		for _, value := range values {
			if value.ExperimentID == experimentID &&
				value.Timestamp.After(timeRange.Start) &&
				value.Timestamp.Before(timeRange.End) &&
				segmentUsers[value.UserID] {
				segmentValues = append(segmentValues, value)
			}
		}
	}

	// Calculate variant metrics for this segment
	variantMetrics := make(map[string][]MetricSummary)
	for variantID := range a.getExperimentVariants(experimentID) {
		variantValues := a.filterByVariant(segmentValues, variantID)
		variantMetrics[variantID] = a.calculateMetricSummaries(variantValues)
	}

	// Generate insights
	insights := a.generateSegmentInsights(segment, variantMetrics)

	return SegmentAnalysis{
		SegmentID:      segmentID,
		SegmentName:    segment.Name,
		UserCount:      int64(len(segmentUsers)),
		VariantMetrics: variantMetrics,
		Insights:       insights,
	}
}

// getUsersInSegment gets all users that match the segment criteria
func (a *AnalyticsEngine) getUsersInSegment(segmentID string) map[string]bool {
	segment := a.segments[segmentID]
	if segment == nil {
		return make(map[string]bool)
	}

	users := make(map[string]bool)

	// This would need access to user attributes in a real implementation
	// For now, we'll return all users as a placeholder
	for _, assignment := range a.assignments {
		users[assignment.UserID] = true
	}

	return users
}

// generateSegmentInsights generates insights for a segment analysis
func (a *AnalyticsEngine) generateSegmentInsights(segment *Segment, variantMetrics map[string][]MetricSummary) []string {
	var insights []string

	// Find the best performing variant for primary metrics
	for metricID, metric := range a.metrics {
		if !metric.IsPrimary {
			continue
		}

		bestVariant := ""
		bestValue := -math.Inf(1)

		for variantID, metrics := range variantMetrics {
			for _, metricSummary := range metrics {
				if metricSummary.MetricID == metricID && metricSummary.Mean > bestValue {
					bestValue = metricSummary.Mean
					bestVariant = variantID
				}
			}
		}

		if bestVariant != "" {
			insights = append(insights, fmt.Sprintf(
				"For %s segment, variant %s performs best on %s with value %.2f",
				segment.Name, bestVariant, metric.Name, bestValue))
		}
	}

	return insights
}

// ExportAnalytics exports analytics data in various formats
func (a *AnalyticsEngine) ExportAnalytics(ctx context.Context, analytics *ExperimentAnalytics, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(analytics, "", "  ")
	case "csv":
		return a.exportToCSV(analytics)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToCSV exports analytics data to CSV format
func (a *AnalyticsEngine) exportToCSV(analytics *ExperimentAnalytics) ([]byte, error) {
	// Implement CSV export logic
	// This is a placeholder implementation
	csv := "Experiment,Variant,Metric,Count,Mean,StandardDev\n"

	for variantID, metrics := range analytics.VariantMetrics {
		for _, metric := range metrics {
			csv += fmt.Sprintf("%s,%s,%s,%d,%.2f,%.2f\n",
				analytics.ExperimentID, variantID, metric.MetricName,
				metric.Count, metric.Mean, metric.StandardDev)
		}
	}

	return []byte(csv), nil
}
