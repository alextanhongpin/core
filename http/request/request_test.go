package request_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/alextanhongpin/errors/validator"
	"github.com/alextanhongpin/testdump/jsondump"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

type loginRequest struct {
	Email string `json:"email"`
}

func (req *loginRequest) Validate() error {
	return validator.Map(map[string]error{
		"email": validator.Required(req.Email, validator.Assert(strings.Contains(req.Email, "@"), "The email is invalid")),
	})
}

func TestBody(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		body := loginRequest{
			Email: "john.appleseed@mail.com",
		}
		b, err := json.Marshal(body)
		is := assert.New(t)
		is.NoError(err)

		r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
		var req loginRequest

		err = request.DecodeJSON(r, &req)
		is.NoError(err)
		is.Empty(cmp.Diff(req, body))
	})

	t.Run("invalid", func(t *testing.T) {
		body := loginRequest{
			Email: "john.doe",
		}
		b, err := json.Marshal(body)
		is := assert.New(t)
		is.NoError(err)

		r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
		var req loginRequest

		err = request.DecodeJSON(r, &req)
		is.Error(err)
		is.Empty(cmp.Diff(req, body))
		jsondump.Dump(t, err)
	})
}

func TestBodyInvalid(t *testing.T) {
	b := []byte(`<HTML>`)
	r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))

	var req loginRequest
	err := request.DecodeJSON(r, &req)
	is := assert.New(t)

	var bodyErr *request.BodyError
	is.ErrorAs(err, &bodyErr)
	is.True(bytes.Equal(b, bodyErr.Body))
	t.Log(string(bodyErr.Body))
}
