package httputil_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/httputil"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestParseAuthHeader(t *testing.T) {
	tests := map[string]struct {
		auth string
		ok   bool
		want string
	}{
		"success":   {"Bearer xyz", true, "xyz"},
		"empty":     {"", false, ""},
		"no bearer": {"xyz", false, ""},
		"no token":  {"Bearer", false, ""},
	}

	for name, ts := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/userinfo", nil)
			r.Header.Set("Authorization", ts.auth)

			token, ok := httputil.BearerAuth(r)
			is := assert.New(t)
			is.Equal(ts.ok, ok)
			is.Equal(ts.want, token)
		})
	}
}

func TestSignAndVerifyJWT(t *testing.T) {
	secret := []byte("secret")
	token, err := httputil.SignJWT(secret, jwt.MapClaims{
		"sub": "john.appleseed@mail.com",
	}, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := httputil.VerifyJWT(secret, token)
	is := assert.New(t)
	is.Nil(err)
	is.Equal("john.appleseed@mail.com", claims["sub"])
}
