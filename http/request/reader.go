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
// Read reads at most MaxBodySize bytes. A larger body fails with
// ErrBodyTooLarge; r.Body then still yields the complete original stream.
func Read(r *http.Request) ([]byte, error) {
	b, err := readAtMost(r, MaxBodySize)
	if err != nil {
		return nil, err
	}

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
// Like Read, it fails with ErrBodyTooLarge for bodies over MaxBodySize.
func Clone(r *http.Request) (*http.Request, error) {
	b, err := Read(r)
	if err != nil {
		return nil, err
	}

	rc := r.Clone(r.Context())
	rc.Body = io.NopCloser(bytes.NewBuffer(b))

	return rc, nil
}

// Parse decodes the JSON body into a new T. It is DecodeJSON for a value
// type, with the same size limit, Bind/Validate hooks and options.
func Parse[T any](r *http.Request, opts ...DecodeOption) (T, error) {
	var v T
	if err := DecodeJSON(r, &v, opts...); err != nil {
		var zero T
		return zero, err
	}

	return v, nil
}
