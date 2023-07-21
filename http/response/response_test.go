package response_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/alextanhongpin/core/test/testutil"
)

func TestJSONError(t *testing.T) {
	dumpError := func(t *testing.T, err error) {
		t.Helper()

		r := httptest.NewRequest("GET", "/user/1", nil)
		h := func(w http.ResponseWriter, r *http.Request) {
			response.JSONError(w, err)
		}
		testutil.DumpHTTPHandler(t, r, h)
	}

	t.Run("known error", func(t *testing.T) {
		dumpError(t, response.ErrBadRequest)
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
		r := httptest.NewRequest("GET", "/users", nil)
		h := func(w http.ResponseWriter, r *http.Request) {
			payload := response.
				OK([]user{
					{ID: "user-1", Name: "Alice"},
					{ID: "user-2", Name: "Bob"},
				}).
				WithLinks(&response.Links{
					Prev: "prev-link",
					Next: "next-link",
				}).
				WithMeta(&response.Meta{
					"count": 2,
				})

			response.JSON(w, payload, http.StatusOK)
		}

		testutil.DumpHTTPHandler(t, r, h)
	})

	t.Run("failed", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/users", nil)
		h := func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, map[string]any{
				"bad_number": json.Number("1.5x"),
			}, http.StatusOK)
		}

		testutil.DumpHTTPHandler(t, r, h)
	})
}
