package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alextanhongpin/core/ab"
)

//go:embed static/*
var static embed.FS

type apiServer struct {
	e  *ab.BanditEngine
	mu sync.Mutex
}

var (
	configMgr *ab.ConfigManager
)

func main() {
	var storage ab.BanditStorage
	// Optionally switch storage via query param or env
	// For demo, use ?storage=memory or ?storage=inmemory (default)
	storageType := "inmemory"
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-storage=memory" || arg == "-storage=inmemory" {
				storageType = "inmemory"
			}
			// Add more storage types here (e.g., redis, sql)
		}
	}
	if storageType == "inmemory" {
		storage = ab.NewInMemoryBanditStorage()
		log.Println("Using in-memory bandit storage")
	} else {
		log.Fatalf("Unknown storage type: %s", storageType)
	}

	provider := ab.NewInMemoryConfigProvider()
	configMgr = ab.NewConfigManager(provider)

	s := &apiServer{e: ab.NewBanditEngineWithStorage(storage, nil)}
	http.HandleFunc("/api/experiments", s.handleExperiments)
	http.HandleFunc("/api/experiments/", s.handleExperiment)
	http.HandleFunc("/api/metrics", s.handleMetrics)
	http.HandleFunc("/api/config/feature-flags", handleFeatureFlags)
	http.HandleFunc("/api/config/feature-flags/", handleFeatureFlags)
	// Add endpoint to enable a feature flag
	http.HandleFunc("/api/config/feature-flags/enable/", handleEnableFeatureFlag)
	http.HandleFunc("/api/config/feature-flags/disable/", handleDisableFeatureFlag)
	http.HandleFunc("/api/config/experiments", handleExperimentConfigs)
	http.HandleFunc("/api/analytics", s.handleAnalytics)
	subFS, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatalf("failed to create sub FS: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(subFS)))
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *apiServer) handleExperiments(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(s.e.Experiments())
		return
	}
	if r.Method == http.MethodPost {
		var exp ab.BanditExperiment
		if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		s.e.CreateBanditExperiment(&exp)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(exp)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *apiServer) handleExperiment(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := r.URL.Path[len("/api/experiments/"):]
	if id == "" {
		http.Error(w, "missing experiment id", 400)
		return
	}

	switch r.Method {
	case http.MethodGet:
		exp := s.findExperiment(id)
		if exp == nil {
			http.Error(w, "not found", 404)
			return
		}
		json.NewEncoder(w).Encode(exp)
		return
	case http.MethodPut:
		exp := s.findExperiment(id)
		if exp == nil {
			http.Error(w, "not found", 404)
			return
		}
		var update ab.BanditExperiment
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// Only update allowed fields
		exp.Name = update.Name
		exp.Description = update.Description
		exp.Algorithm = update.Algorithm
		exp.Arms = update.Arms
		exp.UpdatedAt = update.UpdatedAt
		json.NewEncoder(w).Encode(exp)
		return
	case http.MethodDelete:
		if s.deleteExperiment(id) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not found", 404)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *apiServer) findExperiment(id string) *ab.BanditExperiment {
	for _, exp := range s.e.Experiments() {
		if exp.ID == id {
			return exp
		}
	}
	return nil
}

func (s *apiServer) deleteExperiment(id string) bool {
	exps := s.e.Experiments()
	for _, exp := range exps {
		if exp.ID == id {
			// Remove from slice and map
			s.e.DeleteExperiment(id)
			return true
		}
	}
	return false
}

func (s *apiServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.e.Metrics())
}

func handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	key := ""
	if r.URL.Path != "/api/config/feature-flags" {
		key = r.URL.Path[len("/api/config/feature-flags/"):]
	}
	switch r.Method {
	case http.MethodGet:
		flags, _ := configMgr.ListFeatureFlags(r.Context())
		json.NewEncoder(w).Encode(flags)
	case http.MethodPost:
		var req struct {
			ID          string                 `json:"id"`
			Key         string                 `json:"key"`
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Enabled     *bool                  `json:"enabled"`
			Rules       []ab.FeatureFlagRule   `json:"rules"`
			Rollout     *ab.RolloutConfig      `json:"rollout"`
			Metadata    map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		id := req.ID
		if id == "" {
			id = req.Key
		}
		if id == "" && req.Name != "" {
			id = req.Name
		}
		if id == "" {
			http.Error(w, "missing id/key/name", 400)
			return
		}
		flag := ab.FeatureFlag{
			ID:          id,
			Name:        req.Name,
			Description: req.Description,
			Enabled:     req.Enabled != nil && *req.Enabled,
			Rules:       req.Rules,
			Rollout:     req.Rollout,
			Metadata:    req.Metadata,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := configMgr.CreateFeatureFlag(r.Context(), &flag); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(flag)
	case http.MethodDelete:
		if key == "" {
			http.Error(w, "missing feature flag key", 400)
			return
		}
		if err := configMgr.DeleteFeatureFlag(r.Context(), key); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler to enable a feature flag by key
func handleEnableFeatureFlag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Path[len("/api/config/feature-flags/enable/"):]
	if key == "" {
		http.Error(w, "missing feature flag key", 400)
		return
	}
	flags, _ := configMgr.ListFeatureFlags(r.Context())
	var flag *ab.FeatureFlag
	for _, f := range flags {
		if f.ID == key {
			flag = f
			break
		}
	}
	if flag == nil {
		http.Error(w, "feature flag not found", 404)
		return
	}
	flag.Enabled = true
	flag.UpdatedAt = time.Now()
	if err := configMgr.DeleteFeatureFlag(r.Context(), flag.ID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := configMgr.CreateFeatureFlag(r.Context(), flag); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	json.NewEncoder(w).Encode(flag)
}

// Handler to disable a feature flag by key
func handleDisableFeatureFlag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Path[len("/api/config/feature-flags/disable/"):]
	if key == "" {
		http.Error(w, "missing feature flag key", 400)
		return
	}
	flags, _ := configMgr.ListFeatureFlags(r.Context())
	var flag *ab.FeatureFlag
	for _, f := range flags {
		if f.ID == key {
			flag = f
			break
		}
	}
	if flag == nil {
		http.Error(w, "feature flag not found", 404)
		return
	}
	flag.Enabled = false
	flag.UpdatedAt = time.Now()
	if err := configMgr.DeleteFeatureFlag(r.Context(), flag.ID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := configMgr.CreateFeatureFlag(r.Context(), flag); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	json.NewEncoder(w).Encode(flag)
}

func handleExperimentConfigs(w http.ResponseWriter, r *http.Request) {
	key := ""
	if r.URL.Path != "/api/config/experiments" {
		key = r.URL.Path[len("/api/config/experiments/"):]
	}
	switch r.Method {
	case http.MethodGet:
		configs, _ := configMgr.ListExperimentConfigs(r.Context())
		json.NewEncoder(w).Encode(configs)
	case http.MethodPost:
		var req struct {
			ExperimentID   string                     `json:"experiment_id"`
			Key            string                     `json:"key"`
			TrafficSplit   map[string]float64         `json:"traffic_split"`
			TargetingRules []ab.TargetingRule         `json:"targeting_rules"`
			SampleSize     *ab.SampleSizeConfig       `json:"sample_size"`
			StoppingRules  []ab.StoppingRule          `json:"stopping_rules"`
			MetricConfig   map[string]ab.MetricConfig `json:"metric_config"`
			QualityControl *ab.QualityControlConfig   `json:"quality_control"`
			CreatedAt      *time.Time                 `json:"created_at"`
			UpdatedAt      *time.Time                 `json:"updated_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		id := req.ExperimentID
		if id == "" {
			id = req.Key
		}
		if id == "" {
			http.Error(w, "missing experiment_id/key", 400)
			return
		}
		cfg := ab.ExperimentConfig{
			ExperimentID:   id,
			TrafficSplit:   req.TrafficSplit,
			TargetingRules: req.TargetingRules,
			SampleSize:     req.SampleSize,
			StoppingRules:  req.StoppingRules,
			MetricConfig:   req.MetricConfig,
			QualityControl: req.QualityControl,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if req.CreatedAt != nil {
			cfg.CreatedAt = *req.CreatedAt
		}
		if req.UpdatedAt != nil {
			cfg.UpdatedAt = *req.UpdatedAt
		}
		if len(cfg.TrafficSplit) == 0 {
			cfg.TrafficSplit = map[string]float64{"A": 50, "B": 50}
		}
		if cfg.MetricConfig == nil {
			cfg.MetricConfig = map[string]ab.MetricConfig{}
		}
		if err := configMgr.CreateExperimentConfig(r.Context(), &cfg); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cfg)
	case http.MethodDelete:
		if key == "" {
			http.Error(w, "missing experiment config key", 400)
			return
		}
		if err := configMgr.DeleteExperimentConfig(r.Context(), key); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *apiServer) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"config": configMgr.GetMetrics(),
		"bandit": s.e.Metrics(),
	}
	json.NewEncoder(w).Encode(resp)
}
