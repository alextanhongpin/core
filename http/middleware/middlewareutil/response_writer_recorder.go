package middlewareutil

import (
	"net/http"
)

type ResponseWriterRecorder struct {
	http.ResponseWriter
	statusCode    int
	body          []byte
	headerWritten bool
}

func NewResponseWriterRecorder(w http.ResponseWriter) *ResponseWriterRecorder {
	return &ResponseWriterRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *ResponseWriterRecorder) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true

}

func (w *ResponseWriterRecorder) Write(b []byte) (int, error) {
	w.headerWritten = true
	w.body = b
	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriterRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *ResponseWriterRecorder) Body() []byte {
	return w.body
}

func (w *ResponseWriterRecorder) StatusCode() int {
	return w.statusCode
}
