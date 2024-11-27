package request_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

type loginRequest struct {
	Email string `json:"email"`
}

func (req *loginRequest) Valid() error {
	if !strings.Contains(req.Email, "@") {
		return errors.New("invalid email")
	}

	return nil
}

func TestBody(t *testing.T) {
	body := loginRequest{
		Email: "john.appleseed@mail.com",
	}
	b, err := json.Marshal(body)
	is := assert.New(t)
	is.Nil(err)

	r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
	var req loginRequest
	is.Nil(request.DecodeJSON(r, &req))
	is.Empty(cmp.Diff(req, body))
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
