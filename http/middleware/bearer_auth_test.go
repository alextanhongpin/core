package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/alextanhongpin/core/http/middleware"
	"github.com/alextanhongpin/testdump/httpdump"
)

func TestBearerAuth(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})

	secret := []byte("secret")
	h = middleware.BearerAuthHandler(h, secret)
	expiredToken := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJqb2huLmFwcGxlc2VlZEBtYWlsLmNvbSIsImV4cCI6MTcyOTk0NjI2OH0.W0JqkWLRgaZ7idhjU4pg3t3CibqLuN7ymBYWEKrSXbQ"

	token, err := httputil.SignJWT(secret, httputil.Claims{
		Subject: "john.appleseed@mail.com",
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)

		httpdump.Handler(t, h, httpdump.IgnoreRequestHeaders("Authorization")).ServeHTTP(w, r)
	})

	t.Run("expired", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", expiredToken)

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("no token", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("no token but required", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		httpdump.Handler(t, middleware.RequireAuth(h)).ServeHTTP(w, r)
	})
}
