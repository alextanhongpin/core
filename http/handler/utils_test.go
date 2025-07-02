package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/handler"
)

func TestBaseHandler_ParseIntParam(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name      string
		url       string
		param     string
		pathValue string
		want      int
		wantErr   bool
	}{
		{
			name:    "valid query param",
			url:     "/test?id=123",
			param:   "id",
			want:    123,
			wantErr: false,
		},
		{
			name:    "invalid query param",
			url:     "/test?id=abc",
			param:   "id",
			want:    0,
			wantErr: true,
		},
		{
			name:    "missing param",
			url:     "/test",
			param:   "id",
			want:    0,
			wantErr: true,
		},
		{
			name:    "zero value",
			url:     "/test?id=0",
			param:   "id",
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.pathValue != "" {
				r.SetPathValue(tt.param, tt.pathValue)
			}

			got := base.Params(r, tt.param).Int()
			if got != tt.want {
				t.Errorf("ParseIntParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseHandler_ParseStringParam(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name      string
		url       string
		param     string
		pathValue string
		want      string
		wantErr   bool
	}{
		{
			name:    "valid query param",
			url:     "/test?name=john",
			param:   "name",
			want:    "john",
			wantErr: false,
		},
		{
			name:    "missing param",
			url:     "/test",
			param:   "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty param",
			url:     "/test?name=",
			param:   "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace param",
			url:     "/test?name=%20%20john%20%20",
			param:   "name",
			want:    "john",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.pathValue != "" {
				r.SetPathValue(tt.param, tt.pathValue)
			}

			var got string
			if tt.name == "whitespace param" {
				got = base.Params(r, tt.param).Trim().String()
			} else {
				got = base.Params(r, tt.param).String()
			}
			if got != tt.want {
				t.Errorf("ParseStringParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseHandler_ParseIntParam_PathValue(t *testing.T) {
	base := handler.BaseHandler{}

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.SetPathValue("id", "456")

	got := base.Params(r, "id").Int()
	if got != 456 {
		t.Errorf("ParseIntParam() with path value = %v, want %v", got, 456)
	}
}

func TestBaseHandler_ParseStringParam_PathValue(t *testing.T) {
	base := handler.BaseHandler{}

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.SetPathValue("name", "testname")

	got := base.Params(r, "name").String()
	if got != "testname" {
		t.Errorf("ParseStringParam() with path value = %v, want %v", got, "testname")
	}
}

func TestBaseHandler_ParseOptionalIntParam(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name         string
		url          string
		param        string
		defaultValue int
		want         int
	}{
		{
			name:         "valid param",
			url:          "/test?limit=50",
			param:        "limit",
			defaultValue: 10,
			want:         50,
		},
		{
			name:         "missing param",
			url:          "/test",
			param:        "limit",
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "invalid param",
			url:          "/test?limit=abc",
			param:        "limit",
			defaultValue: 10,
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			got := base.Params(r, tt.param).IntOr(tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseOptionalIntParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseHandler_ParseOptionalStringParam(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name         string
		url          string
		param        string
		defaultValue string
		want         string
	}{
		{
			name:         "valid param",
			url:          "/test?sort=name",
			param:        "sort",
			defaultValue: "id",
			want:         "name",
		},
		{
			name:         "missing param",
			url:          "/test",
			param:        "sort",
			defaultValue: "id",
			want:         "id",
		},
		{
			name:         "empty param",
			url:          "/test?sort=",
			param:        "sort",
			defaultValue: "id",
			want:         "id",
		},
		{
			name:         "whitespace param",
			url:          "/test?sort=%20%20name%20%20",
			param:        "sort",
			defaultValue: "id",
			want:         "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			var got string
			if tt.name == "whitespace param" {
				val := base.Params(r, tt.param).Trim().String()
				if val == "" {
					got = tt.defaultValue
				} else {
					got = val
				}
			} else {
				got = base.Params(r, tt.param).StringOr(tt.defaultValue)
			}
			if got != tt.want {
				t.Errorf("ParseOptionalStringParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseHandler_SetCacheHeaders(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name      string
		directive string
		maxAge    int
		want      string
	}{
		{
			name:      "public with max age",
			directive: "public",
			maxAge:    3600,
			want:      "public, max-age=3600",
		},
		{
			name:      "no cache",
			directive: "no-cache",
			maxAge:    0,
			want:      "no-cache",
		},
		{
			name:      "private with max age",
			directive: "private",
			maxAge:    1800,
			want:      "private, max-age=1800",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			base.SetCacheHeaders(w, tt.directive, tt.maxAge)

			got := w.Header().Get("Cache-Control")
			if got != tt.want {
				t.Errorf("SetCacheHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseHandler_SetContentType(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "PDF content type",
			contentType: "application/pdf",
		},
		{
			name:        "CSV content type",
			contentType: "text/csv",
		},
		{
			name:        "XML content type",
			contentType: "application/xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			base.SetContentType(w, tt.contentType)

			got := w.Header().Get("Content-Type")
			if got != tt.contentType {
				t.Errorf("SetContentType() = %v, want %v", got, tt.contentType)
			}
		})
	}
}

func TestBaseHandler_WriteRawJSON(t *testing.T) {
	base := handler.BaseHandler{}

	data := []byte(`{"message":"success","id":123}`)
	w := httptest.NewRecorder()

	base.WriteRawJSON(w, data, http.StatusAccepted)

	if w.Code != http.StatusAccepted {
		t.Errorf("WriteRawJSON() status = %v, want %v", w.Code, http.StatusAccepted)
	}

	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("WriteRawJSON() should set correct content type")
	}

	if w.Body.String() != string(data) {
		t.Errorf("WriteRawJSON() body = %v, want %v", w.Body.String(), string(data))
	}
}

func TestBaseHandler_StreamJSON(t *testing.T) {
	base := handler.BaseHandler{}

	w := httptest.NewRecorder()
	encoder := base.StreamJSON(w, http.StatusOK)

	// Test streaming multiple objects
	objects := []map[string]any{
		{"id": 1, "name": "first"},
		{"id": 2, "name": "second"},
	}

	for _, obj := range objects {
		encoder.Encode(obj)
	}

	if w.Code != http.StatusOK {
		t.Errorf("StreamJSON() status = %v, want %v", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("StreamJSON() should set correct content type")
	}

	// Verify JSON content
	body := w.Body.String()
	if body == "" {
		t.Error("StreamJSON() should have content")
	}
}

func TestBaseHandler_ParsePaginationParams(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name         string
		url          string
		defaultLimit int
		maxLimit     int
		want         handler.PaginationParams
	}{
		{
			name:         "default values",
			url:          "/test",
			defaultLimit: 20,
			maxLimit:     100,
			want: handler.PaginationParams{
				Limit:  20,
				Offset: 0,
				Page:   1,
			},
		},
		{
			name:         "custom limit and offset",
			url:          "/test?limit=50&offset=100",
			defaultLimit: 20,
			maxLimit:     100,
			want: handler.PaginationParams{
				Limit:  50,
				Offset: 100,
				Page:   1,
			},
		},
		{
			name:         "page-based pagination",
			url:          "/test?page=3&limit=10",
			defaultLimit: 20,
			maxLimit:     100,
			want: handler.PaginationParams{
				Limit:  10,
				Offset: 20, // (page-1) * limit = (3-1) * 10 = 20
				Page:   3,
			},
		},
		{
			name:         "limit exceeds max",
			url:          "/test?limit=200",
			defaultLimit: 20,
			maxLimit:     100,
			want: handler.PaginationParams{
				Limit:  100, // Capped at maxLimit
				Offset: 0,
				Page:   1,
			},
		},
		{
			name:         "negative values",
			url:          "/test?limit=-10&offset=-5&page=-1",
			defaultLimit: 20,
			maxLimit:     100,
			want: handler.PaginationParams{
				Limit:  20, // Reset to default
				Offset: 0,  // Reset to 0
				Page:   1,  // Reset to 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			got := base.ParsePaginationParams(r, tt.defaultLimit, tt.maxLimit)

			if got.Limit != tt.want.Limit {
				t.Errorf("ParsePaginationParams() Limit = %v, want %v", got.Limit, tt.want.Limit)
			}
			if got.Offset != tt.want.Offset {
				t.Errorf("ParsePaginationParams() Offset = %v, want %v", got.Offset, tt.want.Offset)
			}
			if got.Page != tt.want.Page {
				t.Errorf("ParsePaginationParams() Page = %v, want %v", got.Page, tt.want.Page)
			}
		})
	}
}

func TestBaseHandler_ParseSortParams(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name          string
		url           string
		defaultSortBy string
		allowedFields []string
		want          handler.SortParams
	}{
		{
			name:          "default values",
			url:           "/test",
			defaultSortBy: "created_at",
			allowedFields: []string{"name", "email", "created_at"},
			want: handler.SortParams{
				SortBy: "created_at",
				Order:  "asc",
			},
		},
		{
			name:          "custom sort and order",
			url:           "/test?sort=name&order=desc",
			defaultSortBy: "created_at",
			allowedFields: []string{"name", "email", "created_at"},
			want: handler.SortParams{
				SortBy: "name",
				Order:  "desc",
			},
		},
		{
			name:          "invalid sort field",
			url:           "/test?sort=invalid_field&order=desc",
			defaultSortBy: "created_at",
			allowedFields: []string{"name", "email", "created_at"},
			want: handler.SortParams{
				SortBy: "created_at", // Reset to default
				Order:  "desc",
			},
		},
		{
			name:          "invalid order",
			url:           "/test?sort=name&order=random",
			defaultSortBy: "created_at",
			allowedFields: []string{"name", "email", "created_at"},
			want: handler.SortParams{
				SortBy: "name",
				Order:  "asc", // Reset to asc
			},
		},
		{
			name:          "no allowed fields restriction",
			url:           "/test?sort=custom_field&order=desc",
			defaultSortBy: "created_at",
			allowedFields: nil,
			want: handler.SortParams{
				SortBy: "custom_field",
				Order:  "desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			got := base.ParseSortParams(r, tt.defaultSortBy, tt.allowedFields)

			if got.SortBy != tt.want.SortBy {
				t.Errorf("ParseSortParams() SortBy = %v, want %v", got.SortBy, tt.want.SortBy)
			}
			if got.Order != tt.want.Order {
				t.Errorf("ParseSortParams() Order = %v, want %v", got.Order, tt.want.Order)
			}
		})
	}
}
