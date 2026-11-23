package middleware_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/middleware"
)

func TestCORSWildcardWithCredentialsDoesNotReflectOrigin(t *testing.T) {
	// Regression: "*" plus AllowCredentials echoed any Origin with
	// Access-Control-Allow-Credentials, letting every site make credentialed
	// requests.
	h := middleware.CORS(middleware.CORSConfig{AllowCredentials: true})(http.NotFoundHandler())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func TestCORSVaryOrigin(t *testing.T) {
	// Regression: reflected origins were sent without Vary: Origin, so shared
	// caches could serve one origin's response to another.
	h := middleware.CORS(middleware.CORSConfig{AllowedOrigins: []string{"https://a.example"}})(http.NotFoundHandler())

	for _, origin := range []string{"https://a.example", "https://b.example", ""} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if got := w.Header().Get("Vary"); got != "Origin" {
			t.Errorf("origin %q: Vary = %q, want Origin", origin, got)
		}
	}
}

func TestCORSDisallowedOriginGetsNoCORSHeaders(t *testing.T) {
	h := middleware.CORS(middleware.CORSConfig{AllowedOrigins: []string{"https://a.example"}})(http.NotFoundHandler())

	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.Header.Set("Origin", "https://b.example")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want the handler's 404", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("Access-Control-Allow-Methods = %q, want empty", got)
	}
}

func TestCORSPlainOptionsReachesHandler(t *testing.T) {
	// Regression: every OPTIONS request was answered with 204, hiding
	// handlers that implement OPTIONS themselves.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusOK)
	})
	h := middleware.CORS(middleware.DefaultCORSConfig())(next)

	for _, origin := range []string{"", "https://a.example"} {
		r := httptest.NewRequest(http.MethodOptions, "/", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Header().Get("Allow") != "GET" {
			t.Errorf("origin %q: got %d, want the handler's response", origin, w.Code)
		}
	}
}

func TestRecoveryRepanicsErrAbortHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := middleware.Recovery(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if got := recover(); got != http.ErrAbortHandler {
			t.Fatalf("recover() = %v, want http.ErrAbortHandler", got)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
