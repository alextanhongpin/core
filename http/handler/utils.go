// Package handler provides a base controller for HTTP handlers with common functionality.
package handler

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	"github.com/alextanhongpin/core/http/request"
	"github.com/alextanhongpin/core/http/response"
)

// Additional utility methods for the BaseHandler that provide convenient access
// to request parsing and parameter handling functionality.

// QueryValue extracts and returns a query parameter value from the HTTP request.
//
// This is a convenience method that wraps request.QueryValue, providing
// access to the rich Value type functionality for type conversion and validation.
//
// Parameters:
//   - r: The HTTP request to extract the parameter from
//   - param: The name of the query parameter
//
// Returns:
//   - A request.Value that can be converted to various types
//
// Example:
//
//	func (h *MyHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
//		page := h.QueryValue(r, "page").IntOr(1)
//		limit := h.QueryValue(r, "limit").IntOr(10)
//		search := h.QueryValue(r, "search").String()
//		// Use the parameters...
//	}
func (h BaseHandler) QueryValue(r *http.Request, param string) request.Value {
	return request.QueryValue(r, param)
}

// PathValue extracts and returns a path parameter value from the HTTP request.
//
// This is a convenience method that wraps request.PathValue, providing
// access to path parameters defined in route patterns (e.g., "/users/{id}").
//
// Parameters:
//   - r: The HTTP request to extract the parameter from
//   - param: The name of the path parameter
//
// Returns:
//   - A request.Value that can be converted to various types
//
// Example:
//
//	func (h *MyHandler) GetUser(w http.ResponseWriter, r *http.Request) {
//		userID := h.PathValue(r, "id").MustInt()
//		// Use the userID...
//	}
func (h BaseHandler) PathValue(r *http.Request, param string) request.Value {
	return request.PathValue(r, param)
}

func (h BaseHandler) FormValue(r *http.Request, param string) request.Value {
	return request.FormValue(r, param)
}

func (h BaseHandler) Params(r *http.Request, param string) request.Value {
	return cmp.Or(
		h.PathValue(r, param),
		h.QueryValue(r, param),
	)
}

// SetCacheHeaders sets appropriate cache headers for the response.
//
// Example:
//
//	c.SetCacheHeaders(w, "public", 3600) // Cache for 1 hour
//	c.SetCacheHeaders(w, "no-cache", 0)  // No caching
func (h BaseHandler) SetCacheHeaders(w http.ResponseWriter, directive string, maxAge int) {
	if maxAge > 0 {
		w.Header().Set("Cache-Control", directive+", max-age="+strconv.Itoa(maxAge))
	} else {
		w.Header().Set("Cache-Control", directive)
	}
}

// SetContentType sets the content type header.
//
// Example:
//
//	c.SetContentType(w, "application/pdf")
//	c.SetContentType(w, "text/csv")
func (h BaseHandler) SetContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}

// WriteRawJSON writes raw JSON data without additional processing.
// Useful when you already have JSON bytes or want more control.
//
// Example:
//
//	jsonData := []byte(`{"message":"success"}`)
//	c.WriteRawJSON(w, jsonData, http.StatusOK)
func (h BaseHandler) WriteRawJSON(w http.ResponseWriter, data []byte, statusCode int) {
	response.RawJSON(w, data, statusCode)
}

// StreamJSON writes JSON data in streaming fashion for large responses.
// This is useful for large datasets that should be streamed.
//
// Example:
//
//	encoder := c.StreamJSON(w, http.StatusOK)
//	for _, item := range largeDataset {
//		encoder.Encode(item)
//	}
func (h BaseHandler) StreamJSON(w http.ResponseWriter, statusCode int) *json.Encoder {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w)
}

// PaginationParams represents common pagination parameters
type PaginationParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Page   int `json:"page"`
}

// ParsePaginationParams extracts pagination parameters from the request.
//
// Example:
//
//	pagination := c.ParsePaginationParams(r, 20, 100) // default limit 20, max 100
func (h BaseHandler) ParsePaginationParams(r *http.Request, defaultLimit, maxLimit int) PaginationParams {
	limit := min(request.QueryValue(r, "limit").IntOr(defaultLimit), maxLimit)
	if limit < 1 {
		limit = defaultLimit
	}

	offset := max(h.QueryValue(r, "offset").IntOr(0), 0)
	page := max(h.QueryValue(r, "page").IntOr(1), 1)

	// If page is provided, calculate offset
	if r.URL.Query().Has("page") && !r.URL.Query().Has("offset") {
		offset = (page - 1) * limit
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
		Page:   page,
	}
}

// SortParams represents common sorting parameters
type SortParams struct {
	SortBy string `json:"sort_by"`
	Order  string `json:"order"`
}

// ParseSortParams extracts sorting parameters from the request.
//
// Example:
//
//	sort := c.ParseSortParams(r, "created_at", []string{"name", "email", "created_at"})
func (h BaseHandler) ParseSortParams(r *http.Request, defaultSortBy string, allowedFields []string) SortParams {
	sortBy := h.QueryValue(r, "sort").StringOr(defaultSortBy)
	order := h.QueryValue(r, "order").StringOr("asc")

	// Validate sort field
	if len(allowedFields) > 0 {
		if !slices.Contains(allowedFields, sortBy) {
			sortBy = defaultSortBy
		}
	}

	// Validate order
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	return SortParams{
		SortBy: sortBy,
		Order:  order,
	}
}
