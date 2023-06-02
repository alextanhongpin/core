package response_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/alextanhongpin/core/test/testutil"
)

func TestJSONError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "known error",
			err:  response.ErrBadRequest,
		},
		{
			name: "unknown error",
			err:  sql.ErrNoRows,
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/user/1", nil)
			h := func(w http.ResponseWriter, r *http.Request) {
				response.JSONError(w, ts.err)
			}
			testutil.DumpHTTP(t, r, h)
		})
	}
}

func TestJSON(t *testing.T) {
	type credentials struct {
		AccessToken string `json:"accessToken"`
	}

	r := httptest.NewRequest("GET", "/user/1", nil)
	h := func(w http.ResponseWriter, r *http.Request) {
		payload := response.Payload[credentials]{
			Data: &credentials{
				AccessToken: "xyz",
			},
			Links: &response.Links{
				Prev: "prev-link",
				Next: "next-link",
			},
		}

		response.JSON(w, payload, http.StatusOK)
	}
	testutil.DumpHTTP(t, r, h)
}
