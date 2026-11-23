package requestid_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/requestid"
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

	tests := map[string]struct {
		header string
		want   string
	}{
		"new": {"", "random-token"},
		"old": {"abc", "abc"},
		// Regression: any client value was trusted and echoed verbatim.
		"too long":      {strings.Repeat("a", requestid.MaxLength+1), "random-token"},
		"max length":    {strings.Repeat("a", requestid.MaxLength), strings.Repeat("a", requestid.MaxLength)},
		"space":         {"abc def", "random-token"},
		"non-ascii":     {"abcé", "random-token"},
		"control chars": {"abc\x7f", "random-token"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				r.Header.Set("X-Request-Id", tc.header)
			}
			h.ServeHTTP(w, r)

			if got := r.Header.Get("X-Request-Id"); got != tc.header {
				t.Fatalf("caller's request header changed to %q", got)
			}
			if got := w.Header().Get("X-Request-Id"); got != tc.want {
				t.Fatalf("X-Request-Id = %q, want %q", got, tc.want)
			}
			if w.Body.String() != "hello world" {
				t.Fatalf("body = %q", w.Body.String())
			}
		})
	}
}
