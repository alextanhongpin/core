package auth_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/auth"
	"github.com/alextanhongpin/testdump/httpdump"
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

	ctx := context.Background()
	ctx = auth.LoggerContext.WithValue(ctx, slog.New(slog.NewTextHandler(os.Stdout, nil)))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		r = r.WithContext(ctx)

		httpdump.Handler(t, h, httpdump.IgnoreRequestHeaders("Authorization")).ServeHTTP(w, r)
	})

	t.Run("expired", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(ctx)
		r.Header.Set("Authorization", expiredToken)

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("no token", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(ctx)

		httpdump.Handler(t, h).ServeHTTP(w, r)
	})

	t.Run("no token but required", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(ctx)

		httpdump.Handler(t, auth.RequireBearerHandler(h)).ServeHTTP(w, r)
	})
}
