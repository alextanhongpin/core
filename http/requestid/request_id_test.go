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
		reqID, ok := requestid.Context.Value(r.Context())
		if !ok {
			t.Error("request id not found in context")
		}
		if reqID != w.Header().Get("X-Request-Id") {
			t.Errorf("unexpected request id: %s", reqID)
		}

		fmt.Fprint(w, "hello world")
	})
	h = requestid.Handler(h, "X-Request-Id", func() string {
		return "random-token"
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
