package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/auth"
	"github.com/alextanhongpin/testdump/httpdump"
)

func TestBasicAuth(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})
	h = auth.BasicHandler(h, map[string]string{
		"john": "123456",
	})

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123456")

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("john", "123")

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})
}
