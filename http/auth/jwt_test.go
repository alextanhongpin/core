package auth_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/auth"
	"github.com/stretchr/testify/assert"
)

func TestBearerAuth(t *testing.T) {
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

			token, ok := auth.BearerAuth(r)
			is := assert.New(t)
			is.Equal(ts.ok, ok)
			is.Equal(ts.want, token)
		})
	}
}

func TestSignAndVerifyJWT(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		jwt := auth.NewJWT([]byte("secret"))
		token, err := jwt.Sign(auth.Claims{
			Subject: "john.appleseed@mail.com",
		}, 1*time.Hour)
		if err != nil {
			t.Fatal(err)
		}

		claims, err := jwt.Verify(token)
		is := assert.New(t)
		is.Nil(err)
		is.Equal("john.appleseed@mail.com", claims.Subject)
	})

	t.Run("expired", func(t *testing.T) {
		jwt := auth.NewJWT([]byte("secret"))
		token, err := jwt.Sign(auth.Claims{
			Subject: "john.appleseed@mail.com",
		}, -1*time.Hour)
		if err != nil {
			t.Fatal(err)
		}

		_, err = jwt.Verify(token)
		is := assert.New(t)
		is.ErrorIs(err, auth.ErrTokenInvalid)
	})
}
