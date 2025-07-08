package ab

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// RecommendationType represents the type of recommendation algorithm
type RecommendationType string

const (
	RecommendationContentBased  RecommendationType = "content_based"
	RecommendationCollaborative RecommendationType = "collaborative"
	RecommendationHybrid        RecommendationType = "hybrid"
	RecommendationPopularity    RecommendationType = "popularity"
	RecommendationTrending      RecommendationType = "trending"
)

// Item represents an item that can be recommended
type Item struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Category   string                 `json:"category"`
	Tags       []string               `json:"tags"`
	Features   map[string]float64     `json:"features"`    // Feature vector for content-based filtering
	Popularity float64                `json:"popularity"`  // Popularity score
	TrendScore float64                `json:"trend_score"` // Trending score
	CreatedAt  time.Time              `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// User represents a user in the recommendation system
type User struct {
	ID           string             `json:"id"`
	Preferences  map[string]float64 `json:"preferences"` // User preference vector
	Demographics map[string]string  `json:"demographics"`
	Segments     []string           `json:"segments"`
	CreatedAt    time.Time          `json:"created_at"`
}

// Interaction represents a user-item interaction
type Interaction struct {
	UserID        string                 `json:"user_id"`
	ItemID        string                 `json:"item_id"`
	Type          string                 `json:"type"`           // view, click, purchase, like, etc.
	Rating        float64                `json:"rating"`         // explicit rating if available
	ImplicitScore float64                `json:"implicit_score"` // implicit rating derived from behavior
	Timestamp     time.Time              `json:"timestamp"`
	Context       map[string]interface{} `json:"context"` // contextual information
}

// Recommendation represents a recommended item for a user
type Recommendation struct {
	UserID      string                 `json:"user_id"`
	ItemID      string                 `json:"item_id"`
	Score       float64                `json:"score"`
	Reason      string                 `json:"reason"`
	Algorithm   RecommendationType     `json:"algorithm"`
	Context     map[string]interface{} `json:"context"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// RecommendationStorage defines interface for persisting recommendations, users, items, and interactions
// In production, implement this with Redis, SQL, or other backends
type RecommendationStorage interface {
	SaveUser(user *User) error
	SaveItem(item *Item) error
	SaveInteraction(interaction Interaction) error
	SaveRecommendation(rec Recommendation) error
	ListRecommendations(userID string, limit int) ([]Recommendation, error)
}

// InMemoryRecommendationStorage is a simple in-memory implementation
// Not for production use
type InMemoryRecommendationStorage struct {
	users           map[string]*User
	items           map[string]*Item
	interactions    []Interaction
	recommendations map[string][]Recommendation // userID -> recs
}

func NewInMemoryRecommendationStorage() *InMemoryRecommendationStorage {
	return &InMemoryRecommendationStorage{
		users:           make(map[string]*User),
		items:           make(map[string]*Item),
		interactions:    make([]Interaction, 0),
		recommendations: make(map[string][]Recommendation),
	}
}

func (s *InMemoryRecommendationStorage) SaveUser(user *User) error {
	s.users[user.ID] = user
	return nil
}
func (s *InMemoryRecommendationStorage) SaveItem(item *Item) error {
	s.items[item.ID] = item
	return nil
}
func (s *InMemoryRecommendationStorage) SaveInteraction(interaction Interaction) error {
	s.interactions = append(s.interactions, interaction)
	return nil
}
func (s *InMemoryRecommendationStorage) SaveRecommendation(rec Recommendation) error {
	s.recommendations[rec.UserID] = append(s.recommendations[rec.UserID], rec)
	return nil
}
func (s *InMemoryRecommendationStorage) ListRecommendations(userID string, limit int) ([]Recommendation, error) {
	recs := s.recommendations[userID]
	if len(recs) > limit {
		recs = recs[:limit]
	}
	return recs, nil
}

// RecommendationMetrics tracks operational metrics for recommendations
type RecommendationMetrics struct {
	RecommendationsServed int64
	RecommendationErrors  int64
	RecommendationLatency []float64
}

// RecommendationEngine provides personalized recommendations
type RecommendationEngine struct {
	users        map[string]*User
	items        map[string]*Item
	interactions []Interaction

	// Algorithm weights for hybrid recommendations
	contentWeight       float64
	collaborativeWeight float64
	popularityWeight    float64
	trendingWeight      float64

	storage RecommendationStorage
	metrics RecommendationMetrics
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{
		users:               make(map[string]*User),
		items:               make(map[string]*Item),
		interactions:        make([]Interaction, 0),
		contentWeight:       0.4,
		collaborativeWeight: 0.3,
		popularityWeight:    0.2,
		trendingWeight:      0.1,
		storage:             NewInMemoryRecommendationStorage(),
		metrics:             RecommendationMetrics{},
	}
}

// NewRecommendationEngineWithStorage creates a new recommendation engine with custom storage and metrics
func NewRecommendationEngineWithStorage(storage RecommendationStorage, metrics *RecommendationMetrics) *RecommendationEngine {
	if storage == nil {
		storage = NewInMemoryRecommendationStorage()
	}
	if metrics == nil {
		metrics = &RecommendationMetrics{}
	}
	return &RecommendationEngine{
		storage: storage,
		metrics: *metrics,
	}
}

// AddUser adds a user to the recommendation system
func (r *RecommendationEngine) AddUser(user *User) {
	if user.Preferences == nil {
		user.Preferences = make(map[string]float64)
	}
	if user.Demographics == nil {
		user.Demographics = make(map[string]string)
	}
	r.users[user.ID] = user
	_ = r.storage.SaveUser(user)
}

// AddItem adds an item to the recommendation system
func (r *RecommendationEngine) AddItem(item *Item) {
	if item.Features == nil {
		item.Features = make(map[string]float64)
	}
	if item.Metadata == nil {
		item.Metadata = make(map[string]interface{})
	}
	r.items[item.ID] = item
	_ = r.storage.SaveItem(item)
}

// RecordInteraction records a user-item interaction
func (r *RecommendationEngine) RecordInteraction(interaction Interaction) {
	interaction.Timestamp = time.Now()

	// Calculate implicit score based on interaction type
	switch interaction.Type {
	case "view":
		interaction.ImplicitScore = 1.0
	case "click":
		interaction.ImplicitScore = 2.0
	case "like":
		interaction.ImplicitScore = 3.0
	case "share":
		interaction.ImplicitScore = 4.0
	case "purchase":
		interaction.ImplicitScore = 5.0
	default:
		interaction.ImplicitScore = 1.0
	}

	r.interactions = append(r.interactions, interaction)
	_ = r.storage.SaveInteraction(interaction)

	// Update user preferences and item popularity
	r.updateUserPreferences(interaction)
	r.updateItemMetrics(interaction)
}

// GetRecommendations generates recommendations for a user
func (r *RecommendationEngine) GetRecommendations(ctx context.Context, userID string, count int, algorithm RecommendationType) ([]Recommendation, error) {
	start := time.Now()
	user, exists := r.users[userID]
	if !exists {
		r.metrics.RecommendationErrors++
		return nil, fmt.Errorf("user %s not found", userID)
	}

	var recommendations []Recommendation

	switch algorithm {
	case RecommendationContentBased:
		recommendations = r.getContentBasedRecommendations(user, count)
	case RecommendationCollaborative:
		recommendations = r.getCollaborativeRecommendations(user, count)
	case RecommendationPopularity:
		recommendations = r.getPopularityBasedRecommendations(user, count)
	case RecommendationTrending:
		recommendations = r.getTrendingRecommendations(user, count)
	case RecommendationHybrid:
		recommendations = r.getHybridRecommendations(user, count)
	default:
		return nil, fmt.Errorf("unsupported recommendation algorithm: %s", algorithm)
	}

	// Remove items user has already interacted with
	recommendations = r.filterInteractedItems(userID, recommendations)

	// Limit to requested count
	if len(recommendations) > count {
		recommendations = recommendations[:count]
	}

	// Save recommendations
	for _, rec := range recommendations {
		_ = r.storage.SaveRecommendation(rec)
	}
	r.metrics.RecommendationsServed += int64(len(recommendations))
	latency := time.Since(start).Seconds() * 1000
	r.metrics.RecommendationLatency = append(r.metrics.RecommendationLatency, latency)

	return recommendations, nil
}

// getContentBasedRecommendations generates content-based recommendations
func (r *RecommendationEngine) getContentBasedRecommendations(user *User, count int) []Recommendation {
	var recommendations []Recommendation

	for _, item := range r.items {
		similarity := r.calculateContentSimilarity(user.Preferences, item.Features)

		if similarity > 0 {
			recommendations = append(recommendations, Recommendation{
				UserID:      user.ID,
				ItemID:      item.ID,
				Score:       similarity,
				Reason:      "Based on your preferences",
				Algorithm:   RecommendationContentBased,
				GeneratedAt: time.Now(),
			})
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// getCollaborativeRecommendations generates collaborative filtering recommendations
func (r *RecommendationEngine) getCollaborativeRecommendations(user *User, count int) []Recommendation {
	var recommendations []Recommendation

	// Find similar users
	similarUsers := r.findSimilarUsers(user.ID, 10)

	// Get items liked by similar users
	itemScores := make(map[string]float64)

	for _, similarUser := range similarUsers {
		for _, interaction := range r.interactions {
			if interaction.UserID == similarUser.UserID && interaction.ImplicitScore >= 3.0 {
				itemScores[interaction.ItemID] += similarUser.Similarity * interaction.ImplicitScore
			}
		}
	}

	for itemID, score := range itemScores {
		recommendations = append(recommendations, Recommendation{
			UserID:      user.ID,
			ItemID:      itemID,
			Score:       score,
			Reason:      "Users like you also liked this",
			Algorithm:   RecommendationCollaborative,
			GeneratedAt: time.Now(),
		})
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// getPopularityBasedRecommendations generates popularity-based recommendations
func (r *RecommendationEngine) getPopularityBasedRecommendations(user *User, count int) []Recommendation {
	var recommendations []Recommendation

	// Get items sorted by popularity
	var items []*Item
	for _, item := range r.items {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Popularity > items[j].Popularity
	})

	for _, item := range items {
		recommendations = append(recommendations, Recommendation{
			UserID:      user.ID,
			ItemID:      item.ID,
			Score:       item.Popularity,
			Reason:      "Popular item",
			Algorithm:   RecommendationPopularity,
			GeneratedAt: time.Now(),
		})
	}

	return recommendations
}

// getTrendingRecommendations generates trending recommendations
func (r *RecommendationEngine) getTrendingRecommendations(user *User, count int) []Recommendation {
	var recommendations []Recommendation

	// Calculate trending scores based on recent interactions
	r.updateTrendingScores()

	// Get items sorted by trend score
	var items []*Item
	for _, item := range r.items {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].TrendScore > items[j].TrendScore
	})

	for _, item := range items {
		recommendations = append(recommendations, Recommendation{
			UserID:      user.ID,
			ItemID:      item.ID,
			Score:       item.TrendScore,
			Reason:      "Trending now",
			Algorithm:   RecommendationTrending,
			GeneratedAt: time.Now(),
		})
	}

	return recommendations
}

// getHybridRecommendations combines multiple recommendation algorithms
func (r *RecommendationEngine) getHybridRecommendations(user *User, count int) []Recommendation {
	// Get recommendations from different algorithms
	contentRecs := r.getContentBasedRecommendations(user, count*2)
	collaborativeRecs := r.getCollaborativeRecommendations(user, count*2)
	popularityRecs := r.getPopularityBasedRecommendations(user, count*2)
	trendingRecs := r.getTrendingRecommendations(user, count*2)

	// Combine scores with weights
	combinedScores := make(map[string]float64)
	reasons := make(map[string]string)

	for _, rec := range contentRecs {
		combinedScores[rec.ItemID] += rec.Score * r.contentWeight
		reasons[rec.ItemID] = "Personalized for you"
	}

	for _, rec := range collaborativeRecs {
		combinedScores[rec.ItemID] += rec.Score * r.collaborativeWeight
		if reasons[rec.ItemID] == "" {
			reasons[rec.ItemID] = "Users like you also liked this"
		}
	}

	for _, rec := range popularityRecs {
		combinedScores[rec.ItemID] += rec.Score * r.popularityWeight
		if reasons[rec.ItemID] == "" {
			reasons[rec.ItemID] = "Popular item"
		}
	}

	for _, rec := range trendingRecs {
		combinedScores[rec.ItemID] += rec.Score * r.trendingWeight
		if reasons[rec.ItemID] == "" {
			reasons[rec.ItemID] = "Trending now"
		}
	}

	// Create final recommendations
	var recommendations []Recommendation
	for itemID, score := range combinedScores {
		recommendations = append(recommendations, Recommendation{
			UserID:      user.ID,
			ItemID:      itemID,
			Score:       score,
			Reason:      reasons[itemID],
			Algorithm:   RecommendationHybrid,
			GeneratedAt: time.Now(),
		})
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// calculateContentSimilarity calculates cosine similarity between user preferences and item features
func (r *RecommendationEngine) calculateContentSimilarity(userPrefs, itemFeatures map[string]float64) float64 {
	if len(userPrefs) == 0 || len(itemFeatures) == 0 {
		return 0.0
	}

	var dotProduct, userNorm, itemNorm float64

	// Get all unique features
	features := make(map[string]bool)
	for feature := range userPrefs {
		features[feature] = true
	}
	for feature := range itemFeatures {
		features[feature] = true
	}

	for feature := range features {
		userVal := userPrefs[feature]
		itemVal := itemFeatures[feature]

		dotProduct += userVal * itemVal
		userNorm += userVal * userVal
		itemNorm += itemVal * itemVal
	}

	if userNorm == 0 || itemNorm == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(userNorm) * math.Sqrt(itemNorm))
}

// SimilarUser represents a user similar to the target user
type SimilarUser struct {
	UserID     string  `json:"user_id"`
	Similarity float64 `json:"similarity"`
}

// findSimilarUsers finds users similar to the target user
func (r *RecommendationEngine) findSimilarUsers(targetUserID string, count int) []SimilarUser {
	targetUser, exists := r.users[targetUserID]
	if !exists {
		return nil
	}

	var similarUsers []SimilarUser

	for userID, user := range r.users {
		if userID == targetUserID {
			continue
		}

		similarity := r.calculateUserSimilarity(targetUser, user)
		if similarity > 0 {
			similarUsers = append(similarUsers, SimilarUser{
				UserID:     userID,
				Similarity: similarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(similarUsers, func(i, j int) bool {
		return similarUsers[i].Similarity > similarUsers[j].Similarity
	})

	// Limit to requested count
	if len(similarUsers) > count {
		similarUsers = similarUsers[:count]
	}

	return similarUsers
}

// calculateUserSimilarity calculates similarity between two users based on their interaction patterns
func (r *RecommendationEngine) calculateUserSimilarity(user1, user2 *User) float64 {
	// Get user interaction vectors
	user1Items := r.getUserItemScores(user1.ID)
	user2Items := r.getUserItemScores(user2.ID)

	// Calculate cosine similarity
	return r.calculateCosineSimilarity(user1Items, user2Items)
}

// getUserItemScores gets user's scores for all items they've interacted with
func (r *RecommendationEngine) getUserItemScores(userID string) map[string]float64 {
	scores := make(map[string]float64)

	for _, interaction := range r.interactions {
		if interaction.UserID == userID {
			if interaction.Rating > 0 {
				scores[interaction.ItemID] = interaction.Rating
			} else {
				scores[interaction.ItemID] = interaction.ImplicitScore
			}
		}
	}

	return scores
}

// calculateCosineSimilarity calculates cosine similarity between two score vectors
func (r *RecommendationEngine) calculateCosineSimilarity(scores1, scores2 map[string]float64) float64 {
	if len(scores1) == 0 || len(scores2) == 0 {
		return 0.0
	}

	// Get common items
	commonItems := make(map[string]bool)
	for itemID := range scores1 {
		if _, exists := scores2[itemID]; exists {
			commonItems[itemID] = true
		}
	}

	if len(commonItems) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64

	for itemID := range commonItems {
		score1 := scores1[itemID]
		score2 := scores2[itemID]

		dotProduct += score1 * score2
		norm1 += score1 * score1
		norm2 += score2 * score2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// updateUserPreferences updates user preferences based on interactions
func (r *RecommendationEngine) updateUserPreferences(interaction Interaction) {
	user, exists := r.users[interaction.UserID]
	if !exists {
		return
	}

	item, exists := r.items[interaction.ItemID]
	if !exists {
		return
	}

	// Update user preferences based on item features
	for feature, value := range item.Features {
		currentPref := user.Preferences[feature]
		// Use weighted average to update preferences
		user.Preferences[feature] = (currentPref + value*interaction.ImplicitScore) / 2
	}
}

// updateItemMetrics updates item popularity and other metrics
func (r *RecommendationEngine) updateItemMetrics(interaction Interaction) {
	item, exists := r.items[interaction.ItemID]
	if !exists {
		return
	}

	// Update popularity based on interaction type
	switch interaction.Type {
	case "view":
		item.Popularity += 0.1
	case "click":
		item.Popularity += 0.2
	case "like":
		item.Popularity += 0.5
	case "share":
		item.Popularity += 1.0
	case "purchase":
		item.Popularity += 2.0
	}
}

// updateTrendingScores calculates trending scores based on recent interactions
func (r *RecommendationEngine) updateTrendingScores() {
	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// Reset trending scores
	for _, item := range r.items {
		item.TrendScore = 0
	}

	// Calculate based on recent interactions
	for _, interaction := range r.interactions {
		item, exists := r.items[interaction.ItemID]
		if !exists {
			continue
		}

		// Weight recent interactions more heavily
		var weight float64
		if interaction.Timestamp.After(dayAgo) {
			weight = 2.0 // Last 24 hours
		} else if interaction.Timestamp.After(weekAgo) {
			weight = 1.0 // Last week
		} else {
			weight = 0.1 // Older interactions
		}

		item.TrendScore += interaction.ImplicitScore * weight
	}
}

// filterInteractedItems removes items the user has already interacted with
func (r *RecommendationEngine) filterInteractedItems(userID string, recommendations []Recommendation) []Recommendation {
	interactedItems := make(map[string]bool)

	for _, interaction := range r.interactions {
		if interaction.UserID == userID {
			interactedItems[interaction.ItemID] = true
		}
	}

	var filtered []Recommendation
	for _, rec := range recommendations {
		if !interactedItems[rec.ItemID] {
			filtered = append(filtered, rec)
		}
	}

	return filtered
}

// SetAlgorithmWeights sets the weights for hybrid recommendation algorithms
func (r *RecommendationEngine) SetAlgorithmWeights(content, collaborative, popularity, trending float64) error {
	total := content + collaborative + popularity + trending
	if math.Abs(total-1.0) > 0.001 {
		return fmt.Errorf("weights must sum to 1.0, got %f", total)
	}

	r.contentWeight = content
	r.collaborativeWeight = collaborative
	r.popularityWeight = popularity
	r.trendingWeight = trending

	return nil
}
