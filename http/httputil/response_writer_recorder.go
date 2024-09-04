package httputil

import (
	"net/http"
)

type ResponseWriterRecorder struct {
	http.ResponseWriter
	code        int
	body        []byte
	wroteHeader bool
}

func NewResponseWriterRecorder(w http.ResponseWriter) *ResponseWriterRecorder {
	return &ResponseWriterRecorder{
		ResponseWriter: w,
		code:           http.StatusOK,
	}
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
	return w.code
}
