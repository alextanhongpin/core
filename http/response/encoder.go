package response

import (
	"encoding/json"
	"net/http"
)

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func JSON(w http.ResponseWriter, body any, codes ...int) {
	b, err := json.Marshal(body)
	if err != nil {
		Error(w, err)
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
		Error(w, err)
	}
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

func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	body, code := BodyError(err)
	JSON(w, body, code)
}
