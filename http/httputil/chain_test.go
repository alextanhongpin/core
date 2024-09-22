package httputil

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain(t *testing.T) {
	rec := new(recorder)

	h := func(w http.ResponseWriter, r *http.Request) {
		rec.logs = append(rec.logs, "index")
		fmt.Fprint(w, "hello")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	Chain(h, mockMiddleware(rec, "first"), mockMiddleware(rec, "second"))(w, r)

	resp := w.Result()

	t.Run("status code", func(t *testing.T) {
		want := http.StatusOK
		got := resp.StatusCode
		if want != got {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("response body", func(t *testing.T) {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := "hello"
		got := string(b)
		if want != got {
			t.Fatalf("want %s, got %s", want, got)
		}
	})

	t.Run("logs order", func(t *testing.T) {
		want := []string{"before:first", "before:second", "index", "after:second", "after:first"}
		got := rec.logs
		if len(want) != len(got) {
			t.Fatalf("want %d, got %d", len(want), len(got))
		}

		for i, w := range want {
			if w != got[i] {
				t.Fatalf("want %s, got %s", w, got[i])
			}
		}
	})
}

type recorder struct {
	logs []string
}

func mockMiddleware(rec *recorder, msg string) Middleware {
	return Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rec.logs = append(rec.logs, "before:"+msg)

			next(w, r)

			rec.logs = append(rec.logs, "after:"+msg)
		}
	})
}
