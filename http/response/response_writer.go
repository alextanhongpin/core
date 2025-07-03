// Package response provides standardized HTTP response structures and utilities.
package response

import (
	"bytes"
	"net/http"
)

// ResponseWriterRecorder is a wrapper around http.ResponseWriter that captures response data.
//
// This type is useful for middleware that needs to inspect, modify, or log response
// data before it's sent to the client. It can capture the response status code,
// headers, and optionally the response body for processing.
//
// The recorder implements the http.ResponseWriter interface, making it a drop-in
// replacement that can be passed to HTTP handlers while capturing their output.
//
// Example usage:
//
//	func loggingMiddleware(next http.Handler) http.Handler {
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			recorder := response.NewResponseWriterRecorder(w)
//			recorder.SetWriteBody(true) // Capture response body
//
//			next.ServeHTTP(recorder, r)
//
//			// Log response details
//			logger.Info("response",
//				"status", recorder.Code(),
//				"size", len(recorder.Body()))
//		})
//	}
type ResponseWriterRecorder struct {
	http.ResponseWriter
	body        []byte
	code        int
	writeBody   bool
	wroteBody   bool
	wroteHeader bool
}

// NewResponseWriterRecorder creates a new ResponseWriterRecorder that wraps the given ResponseWriter.
//
// If the provided ResponseWriter is already a ResponseWriterRecorder, it returns
// the existing recorder to avoid double-wrapping.
//
// Parameters:
//   - w: The http.ResponseWriter to wrap and record
//
// Returns:
//   - A new ResponseWriterRecorder that captures response data
//
// Example:
//
//	recorder := response.NewResponseWriterRecorder(w)
//	myHandler.ServeHTTP(recorder, r)
//	statusCode := recorder.Code()
func NewResponseWriterRecorder(w http.ResponseWriter) *ResponseWriterRecorder {
	if rw, ok := w.(*ResponseWriterRecorder); ok {
		return rw
	}

	return &ResponseWriterRecorder{
		ResponseWriter: w,
		code:           http.StatusOK,
	}
}

// SetWriteBody configures whether the recorder should capture the response body.
//
// When enabled, the recorder will store a copy of all data written to the response,
// which can be retrieved later using the Body() method. This is useful for logging,
// debugging, or response transformation.
//
// Parameters:
//   - b: true to enable body capturing, false to disable
//
// Note: Enabling body capturing will consume additional memory proportional to
// the response size, so use with caution for large responses.
//
// Example:
//
//	recorder := response.NewResponseWriterRecorder(w)
//	recorder.SetWriteBody(true)
//	// Response body will now be captured
func (w *ResponseWriterRecorder) SetWriteBody(b bool) {
	w.writeBody = b
}

func (w *ResponseWriterRecorder) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriterRecorder) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(w.code)
	}

	if w.writeBody {
		if !w.wroteBody {
			w.body = bytes.Clone(b)
			w.wroteBody = true
		}
	}

	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriterRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *ResponseWriterRecorder) Body() []byte {
	return bytes.Clone(w.body)
}

func (w *ResponseWriterRecorder) StatusCode() int {
	return w.code
}
