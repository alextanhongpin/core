package response_test

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/google/go-cmp/cmp"
)

func TestJSONError(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := response.ErrBadRequest
		response.JSONError(w, err)

		res := w.Result()
		defer res.Body.Close()

		{
			want := http.StatusBadRequest
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
				"code": "bad_request",
				"message": "The input you provided is invalid"
			}
		}`)
			got := b
			cmpJSON(t, want, got)
		}
	})

	t.Run("non-app error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := sql.ErrNoRows

		response.JSONError(w, err)

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
				"code":"internal_server_error",
				"message":"Oops, please try again later"
			}
		}`)
			got := b
			cmpJSON(t, want, got)
		}
	})
}

func TestJSON(t *testing.T) {
	type credentials struct {
		AccessToken string `json:"accessToken"`
	}

	w := httptest.NewRecorder()
	response.JSON(w, response.Payload[credentials]{
		Data: &credentials{
			AccessToken: "xyz",
		},
		Links: &response.Links{
			Prev: "prev-link",
			Next: "next-link",
		},
	}, http.StatusOK)

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
				"accessToken": "xyz"
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
