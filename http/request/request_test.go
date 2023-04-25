package request_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/google/go-cmp/cmp"
)

func TestBody(t *testing.T) {
	type loginRequest struct {
		Email string `json:"email"`
	}

	body := loginRequest{
		Email: "john.appleseed@mail.com",
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
	req, err := request.Body[loginRequest](w, r)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(req, body); diff != "" {
		t.Fatalf("want(+), got(-): %s", diff)
	}
}
