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
	if err != nil {
		t.Error(err)
	}

	r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
	var req loginRequest
	err = request.DecodeJSON(r, &req)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(req, body); diff != "" {
		t.Fatalf("want(+), got(-): %s", diff)
	}
}
