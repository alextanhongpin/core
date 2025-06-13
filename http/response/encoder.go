package response

import (
	"encoding/json"
	"net/http"
)

const maxRecursionDepth = 3

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func OK(w http.ResponseWriter, data any, codes ...int) {
	code := http.StatusOK
	if len(codes) > 0 {
		code = codes[0]
	}

	JSON(w, &Body{
		Data: data,
	}, code)
}

func JSON(w http.ResponseWriter, body any, codes ...int) {
	writeJSON(w, body, 0, codes...)
}

func ErrorJSON(w http.ResponseWriter, err error) {
	writeErrorJSON(w, err, 0)
}

func writeErrorJSON(w http.ResponseWriter, err error, maxDepth int) {
	if err == nil {
		return
	}
	if maxDepth >= maxRecursionDepth {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	body, code := BodyError(err)
	writeJSON(w, body, maxDepth+1, code)
}

func writeJSON(w http.ResponseWriter, body any, maxDepth int, codes ...int) {
	b, err := json.Marshal(body)
	if err != nil {
		writeErrorJSON(w, err, maxDepth+1)

		return
	}

	code := http.StatusOK
	if len(codes) > 0 {
		code = codes[0]
	}

	// This must come before WriteHeader, otherwise the header will not be set correctly.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write(b); err != nil {
		writeErrorJSON(w, err, maxDepth+1)
	}
}
