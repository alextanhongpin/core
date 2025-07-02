package health_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/health"
)

func TestNew(t *testing.T) {
	h := health.New("v1.0.0")
	if h == nil {
		t.Error("Expected handler to be created")
	}
}

func TestLive(t *testing.T) {
	h := health.New("v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	h.Live(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be application/json")
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != health.StatusHealthy {
		t.Errorf("Expected status healthy, got %s", response.Status)
	}

	if response.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", response.Version)
	}

	if response.Uptime <= 0 {
		t.Error("Expected uptime to be positive")
	}
}

func TestReady_NoChecks(t *testing.T) {
	h := health.New("v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != health.StatusHealthy {
		t.Errorf("Expected status healthy, got %s", response.Status)
	}
}

func TestReady_WithHealthyChecks(t *testing.T) {
	h := health.New("v1.0.0")

	h.AddCheck("test", func() health.Check {
		return health.Check{
			Status:  health.StatusHealthy,
			Message: "Test OK",
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != health.StatusHealthy {
		t.Errorf("Expected status healthy, got %s", response.Status)
	}

	if len(response.Checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(response.Checks))
	}

	check, exists := response.Checks["test"]
	if !exists {
		t.Error("Expected test check to exist")
	}

	if check.Status != health.StatusHealthy {
		t.Errorf("Expected check status healthy, got %s", check.Status)
	}

	if check.Message != "Test OK" {
		t.Errorf("Expected message 'Test OK', got %s", check.Message)
	}

	if check.Latency <= 0 {
		t.Error("Expected latency to be measured")
	}
}

func TestReady_WithUnhealthyChecks(t *testing.T) {
	h := health.New("v1.0.0")

	h.AddCheck("failing", func() health.Check {
		return health.Check{
			Status:  health.StatusUnhealthy,
			Message: "Service down",
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != health.StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", response.Status)
	}
}

func TestReady_WithMixedChecks(t *testing.T) {
	h := health.New("v1.0.0")

	h.AddCheck("healthy", func() health.Check {
		return health.Check{
			Status:  health.StatusHealthy,
			Message: "OK",
		}
	})

	h.AddCheck("degraded", func() health.Check {
		return health.Check{
			Status:  health.StatusDegraded,
			Message: "Slow",
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != health.StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", response.Status)
	}
}

func TestDatabaseCheck(t *testing.T) {
	t.Run("healthy database", func(t *testing.T) {
		checker := health.DatabaseCheck(func() error {
			return nil
		})

		check := checker()

		if check.Status != health.StatusHealthy {
			t.Errorf("Expected status healthy, got %s", check.Status)
		}

		if check.Message != "Database connection OK" {
			t.Errorf("Expected message 'Database connection OK', got %s", check.Message)
		}
	})

	t.Run("unhealthy database", func(t *testing.T) {
		checker := health.DatabaseCheck(func() error {
			return errors.New("connection refused")
		})

		check := checker()

		if check.Status != health.StatusUnhealthy {
			t.Errorf("Expected status unhealthy, got %s", check.Status)
		}

		if check.Message != "Database connection failed: connection refused" {
			t.Errorf("Unexpected message: %s", check.Message)
		}
	})
}

func TestHTTPCheck(t *testing.T) {
	t.Run("healthy HTTP service", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		checker := health.HTTPCheck(server.URL, time.Second)
		check := checker()

		if check.Status != health.StatusHealthy {
			t.Errorf("Expected status healthy, got %s", check.Status)
		}

		if check.Message != "HTTP service OK" {
			t.Errorf("Expected message 'HTTP service OK', got %s", check.Message)
		}
	})

	t.Run("unhealthy HTTP service", func(t *testing.T) {
		// Create a test server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		checker := health.HTTPCheck(server.URL, time.Second)
		check := checker()

		if check.Status != health.StatusUnhealthy {
			t.Errorf("Expected status unhealthy, got %s", check.Status)
		}
	})

	t.Run("unreachable HTTP service", func(t *testing.T) {
		checker := health.HTTPCheck("http://localhost:99999", 100*time.Millisecond)
		check := checker()

		if check.Status != health.StatusUnhealthy {
			t.Errorf("Expected status unhealthy, got %s", check.Status)
		}
	})
}

func TestMemoryCheck(t *testing.T) {
	checker := health.MemoryCheck(1024)
	check := checker()

	if check.Status != health.StatusHealthy {
		t.Errorf("Expected status healthy, got %s", check.Status)
	}

	if check.Message != "Memory usage OK" {
		t.Errorf("Expected message 'Memory usage OK', got %s", check.Message)
	}
}
