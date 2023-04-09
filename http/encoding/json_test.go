package encoding_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/go-core-microservice/http/encoding"
	"github.com/alextanhongpin/go-core-microservice/http/types"
	"github.com/google/go-cmp/cmp"
)

type loginRequest struct {
	Email string `json:"email"`
}

func TestDecode(t *testing.T) {
	req := loginRequest{
		Email: "john.appleseed@mail.com",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "/login", bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	}

	res, err := encoding.DecodeJSON[loginRequest](w, r)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(res, req); diff != "" {
		t.Fatalf("want(+), got(-): %s", diff)
	}
}

func TestEncodeError(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := types.ErrUnauthorized

		encoding.EncodeJSONError(w, err)

		res := w.Result()
		defer res.Body.Close()

		{
			want := http.StatusUnauthorized
			got := res.StatusCode
			if want != got {
				t.Fatalf("status code: want %d, got %d", want, got)
			}
		}

		{
			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}
			want := []byte(`{
			"error": {
				"code":"api.unauthorized",
				"message":"You are not logged in."
			}
		}`)
			got := b
			cmpJSON(t, want, got)
		}
	})

	t.Run("non-app error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := sql.ErrNoRows

		encoding.EncodeJSONError(w, err)

		res := w.Result()
		defer res.Body.Close()

		{
			want := http.StatusInternalServerError
			got := res.StatusCode
			if want != got {
				t.Fatalf("status code: want %d, got %d", want, got)
			}
		}

		{
			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}
			want := []byte(`{
			"error": {
				"code":"api.internal",
				"message":"Oops, something went wrong. Please try again later."
			}
		}`)
			got := b
			cmpJSON(t, want, got)
		}
	})
}

func TestEncode(t *testing.T) {
	w := httptest.NewRecorder()
	encoding.EncodeJSON(w, http.StatusOK, types.Result[loginRequest]{
		Data: &loginRequest{
			Email: "john.appleseed@mail.com",
		},
		Links: &types.Links{
			Prev: "prev-link",
			Next: "next-link",
		},
	})

	res := w.Result()
	defer res.Body.Close()

	{
		want := http.StatusOK
		got := res.StatusCode
		if want != got {
			t.Fatalf("status code: want %d, got %d", want, got)
		}
	}

	{
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}

		want := []byte(`{
			"data": {
				"email": "john.appleseed@mail.com"
			},
			"links": {
				"prev": "prev-link",
				"next": "next-link"
			}
		}`)
		got := b
		cmpJSON(t, want, got)
	}
}

func cmpJSON(t *testing.T, lhs, rhs []byte) {
	var lhsMap, rhsMap map[string]any
	if err := json.Unmarshal(lhs, &lhsMap); err != nil {
		t.Error(err)
	}

	if err := json.Unmarshal(rhs, &rhsMap); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(lhsMap, rhsMap); diff != "" {
		t.Fatalf("want(+), got(-): %s", diff)
	}
}
