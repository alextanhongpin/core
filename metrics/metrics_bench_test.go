package metrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func BenchmarkREDTracker(b *testing.B) {
	b.Run("NewRED", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			red := metrics.NewRED("benchmark_service", "benchmark_action")
			red.Done()
		}
	})

	b.Run("REDWithStatusChanges", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			red := metrics.NewRED("benchmark_service", "benchmark_action")
			red.SetStatus("processing")
			red.SetStatus("validating")
			if i%10 == 0 {
				red.Fail()
			}
			red.Done()
		}
	})

	b.Run("ConcurrentRED", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				red := metrics.NewRED("concurrent_service", "concurrent_action")
				red.Done()
			}
		})
	})
}

func BenchmarkInFlightGauge(b *testing.B) {
	b.Run("IncDec", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			metrics.InFlightGauge.Inc()
			metrics.InFlightGauge.Dec()
		}
	})

	b.Run("ConcurrentIncDec", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				metrics.InFlightGauge.Inc()
				metrics.InFlightGauge.Dec()
			}
		})
	})
}

func BenchmarkRequestDurationHandler(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	instrumentedHandler := metrics.RequestDurationHandler("benchmark", handler)

	b.Run("SimpleRequest", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			instrumentedHandler.ServeHTTP(w, req)
		}
	})

	b.Run("ConcurrentRequests", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				instrumentedHandler.ServeHTTP(w, req)
			}
		})
	})
}

func BenchmarkResponseSize(b *testing.B) {
	requests := []*http.Request{
		httptest.NewRequest("GET", "/", nil),
		httptest.NewRequest("POST", "/api/users", nil),
		func() *http.Request {
			r := httptest.NewRequest("PUT", "/api/users/123", nil)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Authorization", "Bearer token")
			return r
		}(),
	}

	b.Run("ObserveResponseSize", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := requests[i%len(requests)]
			metrics.ObserveResponseSize(req)
		}
	})

	b.Run("ConcurrentObserve", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				req := requests[i%len(requests)]
				metrics.ObserveResponseSize(req)
				i++
			}
		})
	})
}

func BenchmarkFullHTTPStack(b *testing.B) {
	// Simulate a real HTTP handler with all metrics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		time.Sleep(100 * time.Microsecond)

		red := metrics.NewRED("api", "handle_request")
		defer red.Done()

		// Simulate occasional errors
		if r.URL.Query().Get("error") == "true" {
			red.Fail()
			http.Error(w, "simulated error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
		metrics.ObserveResponseSize(r)
	})

	instrumentedHandler := metrics.RequestDurationHandler("v1.0", handler)

	b.Run("FullStack", func(b *testing.B) {
		paths := []string{"/api/users", "/api/orders", "/api/products"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := paths[i%len(paths)]
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			instrumentedHandler.ServeHTTP(w, req)
		}
	})

	b.Run("FullStackWithErrors", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errorParam := ""
			if i%10 == 0 { // 10% error rate
				errorParam = "?error=true"
			}

			req := httptest.NewRequest("GET", "/api/test"+errorParam, nil)
			w := httptest.NewRecorder()

			instrumentedHandler.ServeHTTP(w, req)
		}
	})

	b.Run("ConcurrentFullStack", func(b *testing.B) {
		paths := []string{"/api/users", "/api/orders", "/api/products"}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				path := paths[i%len(paths)]
				req := httptest.NewRequest("GET", path, nil)
				w := httptest.NewRecorder()

				instrumentedHandler.ServeHTTP(w, req)
				i++
			}
		})
	})
}

func BenchmarkMetricsRegistration(b *testing.B) {
	b.Run("CreateRegistry", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				prometheus.NewGauge(prometheus.GaugeOpts{
					Name: fmt.Sprintf("test_gauge_%d", i),
					Help: "Test gauge",
				}),
			)
		}
	})

	b.Run("MetricCollection", func(b *testing.B) {
		reg := prometheus.NewRegistry()
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "benchmark_gauge",
			Help: "Benchmark gauge",
		})
		reg.MustRegister(gauge)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gauge.Set(float64(i))
		}
	})
}

// Benchmark memory allocations
func BenchmarkMemoryAllocations(b *testing.B) {
	b.Run("REDTrackerAllocs", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			red := metrics.NewRED("service", "action")
			red.Done()
		}
	})

	b.Run("RequestHandlerAllocs", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})
		instrumentedHandler := metrics.RequestDurationHandler("v1", handler)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			instrumentedHandler.ServeHTTP(w, req)
		}
	})
}
