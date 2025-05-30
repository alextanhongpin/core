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
			response.Error(w, err)
		})
		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	}

	t.Run("known error", func(t *testing.T) {
		dumpError(t, causes.New(codes.BadRequest, "api/bad_request", "The request provided is invalid"))
	})

	t.Run("validation errors", func(t *testing.T) {
		err := response.NewValidationErrors(map[string]string{
			"email": "The email is invalid",
		})
		dumpError(t, err)
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
			data := []user{
				{ID: "user-1", Name: "Alice"},
				{ID: "user-2", Name: "Bob"},
			}

			response.OK(w, data, http.StatusCreated)
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})

	t.Run("failed", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data := map[string]any{
				"bad_number": json.Number("1.5x"),
			}
			response.OK(w, data)
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})

	t.Run("no content", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.NoContent(w)
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})

	t.Run("custom body", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, &response.Body{
				PageInfo: &response.PageInfo{
					HasNextPage: true,
				},
			})
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})

	t.Run("encode", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/users", nil)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := map[string]string{
				"hello": "world",
			}
			response.JSON(w, body, http.StatusAccepted)
		})

		hd := httpdump.Handler(t, h)
		hd.ServeHTTP(wr, r)
	})
}
