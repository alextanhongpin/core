package request_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/go-playground/validator/v10"
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

func TestValidateCustomField(t *testing.T) {
	// Demonstrates how to register custom validation for the validator.
	validate := request.Validator()
	err := validate.RegisterValidation("must_be_foo", func(fl validator.FieldLevel) bool {
		value := fl.Field().Interface().(string)
		return value == "foo"
	})
	if err != nil {
		t.Fatal(err)
	}

	type fooRequest struct {
		Foo string `json:"foo" validate:"must_be_foo"`
	}

	body := fooRequest{}
	b, err := json.Marshal(body)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", bytes.NewReader(b))
	_, err = request.Body[fooRequest](w, r)
	want := "Key: 'fooRequest.Foo' Error:Field validation for 'Foo' failed on the 'must_be_foo' tag"
	got := err.Error()
	if want != got {
		t.Fatalf("want %s, got %s", want, got)
	}
}
