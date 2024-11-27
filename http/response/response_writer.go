package response

import "net/http"

type ResponseWriterRecorder struct {
	http.ResponseWriter
	body        []byte
	code        int
	writeBody   bool
	wroteBody   bool
	wroteHeader bool
}

func NewResponseWriterRecorder(w http.ResponseWriter) *ResponseWriterRecorder {
	if rw, ok := w.(*ResponseWriterRecorder); ok {
		return rw
	}

	return &ResponseWriterRecorder{
		ResponseWriter: w,
		code:           http.StatusOK,
	}
}

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
			w.body = b
			w.wroteBody = true
		}
	}

	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriterRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *ResponseWriterRecorder) Body() ([]byte, bool) {
	return w.body, w.wroteBody
}

func (w *ResponseWriterRecorder) StatusCode() int {
	return w.code
}
