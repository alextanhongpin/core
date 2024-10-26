package middleware_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/middleware"
)

func TestChain(t *testing.T) {
	rec := new(recorder)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.logs = append(rec.logs, "index")
		fmt.Fprint(w, "hello")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.Chain(h, mockMiddleware(rec, "first"), mockMiddleware(rec, "second")).ServeHTTP(w, r)

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

func TestChainFunc(t *testing.T) {
	rec := new(recorder)

	h := func(w http.ResponseWriter, r *http.Request) {
		rec.logs = append(rec.logs, "index")
		fmt.Fprint(w, "hello")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.ChainFunc(h, mockMiddlewareFunc(rec, "first"), mockMiddlewareFunc(rec, "second"))(w, r)

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

func mockMiddleware(rec *recorder, msg string) middleware.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec.logs = append(rec.logs, "before:"+msg)

			h.ServeHTTP(w, r)

			rec.logs = append(rec.logs, "after:"+msg)
		})
	}
}

func mockMiddlewareFunc(rec *recorder, msg string) middleware.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rec.logs = append(rec.logs, "before:"+msg)

			next(w, r)

			rec.logs = append(rec.logs, "after:"+msg)
		}
	}
}
