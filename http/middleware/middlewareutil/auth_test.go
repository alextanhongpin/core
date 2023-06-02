package middlewareutil_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/middleware/middlewareutil"
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
			assert := assert.New(t)
			r := httptest.NewRequest("POST", "/userinfo", nil)
			r.Header.Set("Authorization", ts.auth)

			token, ok := middlewareutil.BearerAuth(r)
			assert.Equal(ts.ok, ok)
			assert.Equal(ts.want, token)
		})
	}
}

func TestSignAndVerifyJWT(t *testing.T) {
	assert := assert.New(t)

	secret := []byte("secret")
	token, err := middlewareutil.SignJWT(secret, jwt.MapClaims{
		"sub": "john.appleseed@mail.com",
	}, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := middlewareutil.VerifyJWT(secret, token)
	assert.Nil(err)
	assert.Equal("john.appleseed@mail.com", claims["sub"])
}
