// Package request provides utilities for HTTP request parsing, validation, and manipulation.
package request

import (
	"bytes"
	"io"
	"net/http"
)

// Read reads the entire request body and restores it for subsequent reads.
//
// This function is useful when you need to read the request body multiple times
// or when middleware needs to inspect the body without consuming it for the
// actual handler. It reads all data from the request body and then replaces
// the body with a new reader containing the same data.
//
// The original request body is consumed and replaced with a new io.ReadCloser
// that contains the same data, allowing the body to be read again.
//
// Parameters:
//   - r: The HTTP request whose body should be read
//
// Returns:
//   - The request body content as a byte slice
//   - An error if reading the body fails
//
// Example:
//
//	// Read body for logging while preserving it for the handler
//	body, err := request.Read(r)
//	if err != nil {
//		return err
//	}
//	logger.Debug("request body", "body", string(body))
//	// The request body is still available for the handler to read
//
// Note: This function loads the entire request body into memory, so it should
// be used with caution for large request bodies to avoid memory issues.
func Read(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

// Clone creates a deep copy of an HTTP request, including its body.
//
// This function is useful when you need to process the same request in multiple
// ways or when middleware needs to modify a request without affecting the
// original. The cloned request has its own copy of the body, allowing both
// the original and cloned requests to be read independently.
//
// The function uses Read() internally to preserve the original request body
// while creating a separate copy for the clone.
//
// Parameters:
//   - r: The HTTP request to clone
//
// Returns:
//   - A new HTTP request that is an independent copy of the original
//   - An error if reading the original request body fails
//
// Example:
//
//	// Clone request for parallel processing
//	clonedReq, err := request.Clone(r)
//	if err != nil {
//		return err
//	}
//
//	// Both original and cloned requests can now be used independently
//	go processOriginal(r)
//	go processClone(clonedReq)
//
// Note: Like Read(), this function loads the entire request body into memory.
func Clone(r *http.Request) (*http.Request, error) {
	b, err := Read(r)
	if err != nil {
		return nil, err
	}

	rc := r.Clone(r.Context())
	rc.Body = io.NopCloser(bytes.NewBuffer(b))

	return rc, nil
}
