// Package response provides standardized HTTP response structures and utilities.
package response

import (
	"bytes"
	"io"
	"net/http"
)

// Read reads the entire response body and restores it for subsequent reads.
//
// This function is useful when you need to read an HTTP response body multiple times
// or when middleware needs to inspect the response body without consuming it for
// downstream processing. It reads all data from the response body and then replaces
// the body with a new reader containing the same data.
//
// The original response body is consumed and replaced with a new io.ReadCloser
// that contains the same data, allowing the body to be read again.
//
// Parameters:
//   - w: The HTTP response whose body should be read
//
// Returns:
//   - The response body content as a byte slice
//   - An error if reading the body fails
//
// Example:
//
//	// Read response body for logging while preserving it for processing
//	body, err := response.Read(resp)
//	if err != nil {
//		return err
//	}
//	logger.Debug("response body", "body", string(body))
//	// The response body is still available for further processing
//
// Note: This function loads the entire response body into memory, so it should
// be used with caution for large response bodies to avoid memory issues.
func Read(w *http.Response) ([]byte, error) {
	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}

	w.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}
