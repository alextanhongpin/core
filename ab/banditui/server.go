package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/alextanhongpin/core/ab"
)

//go:embed static/*
var static embed.FS

type apiServer struct {
	e  *ab.BanditEngine
	mu sync.Mutex
}

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

	s := &apiServer{e: ab.NewBanditEngineWithStorage(storage, nil)}
	http.HandleFunc("/api/experiments", s.handleExperiments)
	http.HandleFunc("/api/experiments/", s.handleExperiment)
	http.HandleFunc("/api/metrics", s.handleMetrics)
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
