package response_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/alextanhongpin/errors/causes"
	"github.com/alextanhongpin/errors/codes"
	"github.com/alextanhongpin/testdump/httpdump"
)

func TestJSONError(t *testing.T) {
	dumpError := func(t *testing.T, err error) {
		t.Helper()

		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/user/1", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wr := response.NewJSONEncoder(w, r)
			defer wr.Flush()

			wr.SetError(err)
		})
		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	}

	t.Run("known error", func(t *testing.T) {
		dumpError(t, causes.New(codes.BadRequest, "api/bad_request", "The request provided is invalid"))
	})

	t.Run("unknown error", func(t *testing.T) {
		dumpError(t, sql.ErrNoRows)
	})
}

func TestJSON(t *testing.T) {
	type user struct {
		ID   string `json:"id"`
		Name string
	}

	t.Run("success", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wr := response.NewJSONEncoder(w, r)
			defer wr.Flush()

			wr.SetData([]user{
				{ID: "user-1", Name: "Alice"},
				{ID: "user-2", Name: "Bob"},
			})
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})

	t.Run("failed", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wr := response.NewJSONEncoder(w, r)
			defer wr.Flush()

			wr.SetData(
				map[string]any{
					"bad_number": json.Number("1.5x"),
				},
				http.StatusCreated,
			)
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})
}
