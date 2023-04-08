package middleware_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/go-core-microservice/http/middleware"
	jwt "github.com/golang-jwt/jwt/v5"
)

func TestParseAuthorizationHeader(t *testing.T) {
	testcases := map[string]struct {
		authHeader string
		wantErr    error
		wantToken  string
	}{
		"success":          {"Bearer xyz", nil, "xyz"},
		"empty":            {"", middleware.ErrAuthorizationHeaderInvalid, ""},
		"no bearer":        {"xyz", middleware.ErrAuthorizationHeaderInvalid, ""},
		"no token":         {"Bearer", middleware.ErrAuthorizationHeaderInvalid, ""},
		"bearer lowercase": {"bearer xyz", middleware.ErrBearerInvalid, ""},
		"invalid bearer":   {"basic xyz", middleware.ErrBearerInvalid, ""},
		"token empty":      {"Bearer ", middleware.ErrTokenMissing, ""},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			token, err := middleware.ParseAuthorizationHeader(tc.authHeader)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("want %s, got %s", tc.wantErr, err)
				}
			}

			if want, got := tc.wantToken, token; want != got {
				t.Errorf("want token %s, got %s", want, got)
			}
		})
	}
}

func TestValidateAuthorizationHeader(t *testing.T) {
	hmacSampleSecret := []byte("secret")

	t.Run("success", func(t *testing.T) {
		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(1 * time.Minute).Unix(), // NOTE: It must be unix.
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString(hmacSampleSecret)
		if err != nil {
			t.Error(err)
		}

		claims, err := middleware.ValidateAuthorizationHeader(hmacSampleSecret, tokenString)
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
		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(-1 * time.Minute).Unix(), // NOTE: It must be unix.
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString(hmacSampleSecret)
		if err != nil {
			t.Error(err)
		}

		_, err = middleware.ValidateAuthorizationHeader(hmacSampleSecret, tokenString)
		if !errors.Is(err, middleware.ErrTokenExpired) {
			t.Errorf("want %v, got %v", middleware.ErrTokenExpired, err)
		}
	})
}
