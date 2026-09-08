package request_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/alextanhongpin/errors/validation"
	"github.com/alextanhongpin/snapshot"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

type loginRequest struct {
	Email string `json:"email"`
}

func (req *loginRequest) Validate() error {
	ve := make(validation.Errors)
	ve.If("email", !strings.Contains(req.Email, "@"), "The email is invalid")
	return ve.Error()
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
		snapshot.New(t).JSON(err)
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
