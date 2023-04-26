package middleware_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/middleware"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestParseAuthHeader(t *testing.T) {
	testcases := map[string]struct {
		authHeader string
		want       bool
		wantToken  string
	}{
		"success":   {"Bearer xyz", true, "xyz"},
		"empty":     {"", false, ""},
		"no bearer": {"xyz", false, ""},
		"no token":  {"Bearer", false, ""},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			token, ok := middleware.ParseAuthHeader(tc.authHeader)
			assert.Equal(tc.want, ok)
			assert.Equal(tc.wantToken, token)
		})
	}
}

func TestValidateAuthHeader(t *testing.T) {
	hmacSampleSecret := []byte("secret")

	t.Run("success", func(t *testing.T) {
		tokenString, err := createToken(hmacSampleSecret, "user-1", time.Now().Add(1*time.Minute))
		if err != nil {
			t.Error(err)
		}

		claims, err := middleware.ValidateAuthHeader(hmacSampleSecret, tokenString)
		if err != nil {
			t.Error(err)
		}

		sub, err := claims.GetSubject()
		if err != nil {
			t.Error(err)
		}

		if want, got := "user-1", sub; want != got {
			t.Errorf("want %s, got %s", want, got)
		}
	})

	t.Run("expired", func(t *testing.T) {
		tokenString, err := createToken(hmacSampleSecret, "user-1", time.Now().Add(-1*time.Minute))
		if err != nil {
			t.Error(err)
		}

		_, err = middleware.ValidateAuthHeader(hmacSampleSecret, tokenString)
		if !errors.Is(err, middleware.ErrTokenExpired) {
			t.Errorf("want %v, got %v", middleware.ErrTokenExpired, err)
		}
	})
}

func TestRequireAuth(t *testing.T) {
	t.Run("token expired", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "hello")
		}

		secret := []byte("secret")

		// Generate a token that expires 1 minute later for 'user-1'
		token, err := createToken(secret, "user-1", time.Now().Add(-1*time.Minute))
		if err != nil {
			t.Error(err)
		}

		// Prepare the mock request and response writer.
		w := httptest.NewRecorder()
		r, err := http.NewRequest("GET", "/private", nil)
		if err != nil {
			t.Error(err)
		}
		r.Header.Set("Authorization", "Bearer "+token)

		// Apply the middleware to the target handler, and call the ServeHTTP
		// so that we can capture the response.
		middleware.RequireAuth(secret)(http.HandlerFunc(handler)).ServeHTTP(w, r)

		res := w.Result()
		defer res.Body.Close()

		// Read the response body.
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}

		// Expected user to be unauthorized because token expired.
		if want, got := 401, res.StatusCode; want != got {
			t.Fatalf("status code: want %d, got %d", want, got)
		}

		// Expected to receive the error envelope with the code and message.
		wantBody := `{"error":{"code":"unauthorized","message":"You are not logged in"}}`
		wantBody += "\n"
		if diff := cmp.Diff(wantBody, string(b)); diff != "" {
			t.Fatalf("want(+), got(-): %s", diff)
		}
	})
}

func createToken(secret []byte, subject string, expiresAt time.Time) (string, error) {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": subject,
		"exp": expiresAt.Unix(), // NOTE: It must be unix.
	})

	// Sign and get the complete encoded token as a string using the secret.
	tokenString, err := token.SignedString(secret)
	return tokenString, err
}
