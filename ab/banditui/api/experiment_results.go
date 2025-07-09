package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alextanhongpin/core/ab"
)

// Assumes you have a global or injected experiment engine
var experimentEngine *ab.ExperimentEngine

func RegisterExperimentResultsAPI(mux *http.ServeMux, engine *ab.ExperimentEngine) {
	experimentEngine = engine
	mux.HandleFunc("/api/experiments/", handleExperimentResults)
}

func handleExperimentResults(w http.ResponseWriter, r *http.Request) {
	// Path: /api/experiments/{id}/results
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "experiments" || parts[3] != "results" {
		http.NotFound(w, r)
		return
	}
	expID := parts[2]
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	results, err := experimentEngine.GetExperimentResults(expID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
