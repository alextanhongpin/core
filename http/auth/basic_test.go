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
		if got, want := w.Header().Get("WWW-Authenticate"), `Basic realm="User Visible Realm"`; got != want {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
		}
	})
}
