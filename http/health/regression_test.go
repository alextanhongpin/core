package health_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/health"
)

func TestAddCheckConcurrentWithReady(t *testing.T) {
	// Regression: AddCheck wrote the map while Ready ranged over it
	// (detected by -race).
	h := health.New("v1")
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() {
			h.AddCheck(fmt.Sprint(i), func() health.Check { return health.Check{Status: health.StatusHealthy} })
		})
		wg.Go(func() {
			h.Ready(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		})
	}
	wg.Wait()
}

func TestReadyUnhealthyWinsOverDegraded(t *testing.T) {
	h := health.New("v1")
	h.AddCheck("a", func() health.Check { return health.Check{Status: health.StatusDegraded} })
	h.AddCheck("b", func() health.Check { return health.Check{Status: health.StatusUnhealthy} })
	h.AddCheck("c", func() health.Check { return health.Check{Status: "bogus"} })

	w := httptest.NewRecorder()
	h.Ready(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestReadyRunsChecksConcurrently(t *testing.T) {
	h := health.New("v1")
	for i := range 5 {
		h.AddCheck(fmt.Sprint(i), func() health.Check {
			time.Sleep(100 * time.Millisecond)
			return health.Check{Status: health.StatusHealthy}
		})
	}

	start := time.Now()
	h.Ready(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if d := time.Since(start); d > 400*time.Millisecond {
		t.Fatalf("Ready took %v; checks ran sequentially", d)
	}
}
