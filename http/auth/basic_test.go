package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/auth"
	"github.com/alextanhongpin/snapshot"
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

		snapshot.HTTP(t, h).ServeHTTP(w, r)
	})

	t.Run("failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123")

		snapshot.HTTP(t, h).ServeHTTP(w, r)
	})
}
