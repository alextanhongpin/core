// Package response provides standardized HTTP response structures and utilities.
//
// This package defines common response formats for JSON APIs, ensuring consistency
// across all endpoints in an application. It follows established patterns for
// API responses including proper error formatting and pagination metadata.
//
// The standard response structure includes:
// - Data field for successful response payload
// - Error field for error details with structured information
// - PageInfo field for pagination metadata
//
// Example usage:
//
//	// Success response
//	response := &response.Body{
//		Data: user,
//	}
//
//	// Error response
//	response := &response.Body{
//		Error: &response.Error{
//			Code:    "validation_failed",
//			Message: "Invalid input data",
//			Errors:  validationErrors,
//		},
//	}
//
//	// Paginated response
//	response := &response.Body{
//		Data: users,
//		PageInfo: &response.PageInfo{
//			HasNextPage: true,
//			HasPrevPage: false,
//			StartCursor: "abc123",
//			EndCursor:   "def456",
//		},
//	}
package response

// PageInfo contains pagination information for paginated API responses.
//
// This structure follows GraphQL Relay specification for cursor-based pagination,
// providing clients with the necessary information to navigate through result sets.
// It supports both cursor-based and offset-based pagination patterns.
type PageInfo struct {
	// HasPrevPage indicates whether there are items before the current page
	HasPrevPage bool `json:"hasPrevPage"`
	// HasNextPage indicates whether there are items after the current page
	HasNextPage bool `json:"hasNextPage"`
	// StartCursor is the cursor of the first item in the current page (for cursor-based pagination)
	StartCursor string `json:"startCursor,omitempty"`
	// EndCursor is the cursor of the last item in the current page (for cursor-based pagination)
	EndCursor string `json:"endCursor,omitempty"`
}

// Body represents the standard API response structure used across all endpoints.
//
// This structure provides a consistent format for all API responses, whether
// successful or containing errors. It follows common API design patterns and
// supports both simple responses and complex paginated results.
//
// Only one of Data or Error should be populated in a given response:
// - For successful operations, populate Data and optionally PageInfo
// - For failed operations, populate Error with detailed error information
type Body struct {
	// Data contains the successful response payload (any valid JSON value)
	Data any `json:"data,omitempty"`
	// Error contains detailed error information when the operation fails
	Error *Error `json:"error,omitempty"`
	// PageInfo contains pagination metadata for list responses
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

// Error represents structured error information in API responses.
//
// This structure provides detailed error information that clients can use
// for error handling, user feedback, and debugging. It supports both
// simple error messages and complex validation errors with field-specific
// details.
type Error struct {
	// Code is a machine-readable error identifier (e.g., "validation_failed", "not_found")
	Code string `json:"code"`
	// Message is a human-readable error description
	Message string `json:"message"`
	// Errors contains field-specific error details (typically for validation errors)
	// The map key is the field name, and the value contains the specific error details
	Errors map[string]any `json:"errors,omitempty"`
}
