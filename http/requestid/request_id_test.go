package requestid_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/requestid"
	"github.com/alextanhongpin/testdump/httpdump"
)

func TestRequestID(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})
	h = requestid.Handler(h, "X-Request-Id", func() (string, error) {
		return "random-token", nil
	})

	t.Run("new", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("old", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Request-Id", "abc")

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})
}
