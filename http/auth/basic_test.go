package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/auth"
)

func TestBasicAuth(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "hello world")
	})
	h = auth.BasicHandler(h, map[string]string{
		"john": auth.HashPasswordSHA256("123456"),
	})

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123456")
		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK || w.Body.String() != "hello world" {
			t.Fatalf("got %d %q, want 200 %q", w.Code, w.Body.String(), "hello world")
		}
	})

	t.Run("failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123")
		h.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
		if got, want := w.Header().Get("WWW-Authenticate"), `Basic realm="User Visible Realm", charset="UTF-8"`; got != want {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("jane", "123456")
		h.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("plain-text credential never matches", func(t *testing.T) {
		h := auth.BasicHandler(http.NotFoundHandler(), map[string]string{"john": "123456"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123456")
		h.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

func TestBasicAuthRealmIsQuoted(t *testing.T) {
	// Regression: the realm was interpolated unescaped, so a quote in it
	// produced a malformed challenge.
	h := auth.BasicHandlerWithConfig(http.NotFoundHandler(), auth.BasicAuthConfig{
		Realm: `a "quoted" \ realm`,
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	want := `Basic realm="a \"quoted\" \\ realm", charset="UTF-8"`
	if got := w.Header().Get("WWW-Authenticate"); got != want {
		t.Fatalf("WWW-Authenticate = %s, want %s", got, want)
	}
}
