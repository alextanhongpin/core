package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Additional utility methods for the BaseHandler

// ParseIntParam extracts and parses an integer parameter from the URL path or query.
// It returns a structured error if the parameter is missing or invalid.
//
// Example:
//
//	userID, err := c.ParseIntParam(r, "id")
//	if err != nil {
//		c.Next(w, r, err)
//		return
//	}
func (h BaseHandler) ParseIntParam(r *http.Request, param string) (int, error) {
	// First try path value (for newer Go versions with routing)
	if value := r.PathValue(param); value != "" {
		id, err := strconv.Atoi(value)
		if err != nil {
			return 0, &ValidationError{
				Field:   param,
				Message: "must be a valid integer",
				Value:   value,
			}
		}
		return id, nil
	}

	// Fallback to query parameter
	value := r.URL.Query().Get(param)
	if value == "" {
		return 0, &ValidationError{
			Field:   param,
			Message: "is required",
		}
	}

	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, &ValidationError{
			Field:   param,
			Message: "must be a valid integer",
			Value:   value,
		}
	}

	return id, nil
}

// ParseStringParam extracts a string parameter from the URL path or query.
// It returns a structured error if the parameter is missing.
//
// Example:
//
//	category, err := c.ParseStringParam(r, "category")
//	if err != nil {
//		c.Next(w, r, err)
//		return
//	}
func (h BaseHandler) ParseStringParam(r *http.Request, param string) (string, error) {
	// First try path value
	if value := r.PathValue(param); value != "" {
		return value, nil
	}

	// Fallback to query parameter
	value := r.URL.Query().Get(param)
	if value == "" {
		return "", &ValidationError{
			Field:   param,
			Message: "is required",
		}
	}

	return strings.TrimSpace(value), nil
}

// ParseOptionalIntParam extracts an optional integer parameter.
// Returns the default value if the parameter is not provided.
//
// Example:
//
//	limit := c.ParseOptionalIntParam(r, "limit", 10)
//	offset := c.ParseOptionalIntParam(r, "offset", 0)
func (h BaseHandler) ParseOptionalIntParam(r *http.Request, param string, defaultValue int) int {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}

	if id, err := strconv.Atoi(value); err == nil {
		return id
	}

	return defaultValue
}

// ParseOptionalStringParam extracts an optional string parameter.
// Returns the default value if the parameter is not provided.
//
// Example:
//
//	sortBy := c.ParseOptionalStringParam(r, "sort", "created_at")
//	order := c.ParseOptionalStringParam(r, "order", "desc")
func (h BaseHandler) ParseOptionalStringParam(r *http.Request, param string, defaultValue string) string {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
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

// ValidationError represents a parameter validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Map returns a map representation for structured error responses
func (e *ValidationError) Map() map[string]any {
	result := map[string]any{
		e.Field: e.Message,
	}
	return map[string]any{
		"errors": result,
	}
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
	limit := h.ParseOptionalIntParam(r, "limit", defaultLimit)
	if limit > maxLimit {
		limit = maxLimit
	}
	if limit < 1 {
		limit = defaultLimit
	}

	offset := h.ParseOptionalIntParam(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	page := h.ParseOptionalIntParam(r, "page", 1)
	if page < 1 {
		page = 1
	}

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
	sortBy := h.ParseOptionalStringParam(r, "sort", defaultSortBy)
	order := h.ParseOptionalStringParam(r, "order", "asc")

	// Validate sort field
	if len(allowedFields) > 0 {
		valid := false
		for _, field := range allowedFields {
			if field == sortBy {
				valid = true
				break
			}
		}
		if !valid {
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
