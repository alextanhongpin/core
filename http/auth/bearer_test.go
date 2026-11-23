package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/auth"
)

func TestBearerHandler(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})

	secret := []byte("secret")
	jwt := auth.NewJWT(secret)
	h = auth.BearerHandler(h, secret)
	expiredToken := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJqb2huLmFwcGxlc2VlZEBtYWlsLmNvbSIsImV4cCI6MTcyOTk0NjI2OH0.W0JqkWLRgaZ7idhjU4pg3t3CibqLuN7ymBYWEKrSXbQ"

	token, err := jwt.Sign(auth.Claims{
		Subject: "john.appleseed@mail.com",
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success", func(t *testing.T) {
		wantOK(t, serve(h, "Bearer "+token))
	})

	t.Run("expired", func(t *testing.T) {
		wantUnauthorized(t, serve(h, expiredToken), `Bearer error="invalid_token"`)
	})

	t.Run("no token", func(t *testing.T) {
		wantOK(t, serve(h, ""))
	})

	t.Run("no token but required", func(t *testing.T) {
		wantUnauthorized(t, serve(auth.RequireBearerHandler(h), ""), `Bearer error="invalid_token"`)
	})
}

func serve(h http.Handler, authz string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if authz != "" {
		r.Header.Set("Authorization", authz)
	}
	h.ServeHTTP(w, r)
	return w
}

func wantOK(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusOK || w.Body.String() != "hello world" {
		t.Fatalf("got %d %q, want 200 %q", w.Code, w.Body.String(), "hello world")
	}
}

func wantUnauthorized(t *testing.T, w *httptest.ResponseRecorder, challenge string) {
	t.Helper()
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got := w.Header().Get("WWW-Authenticate"); got != challenge {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, challenge)
	}
}
