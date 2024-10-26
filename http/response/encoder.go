package response

import (
	"encoding/json"
	"net/http"
)

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func JSON(w http.ResponseWriter, body any, codes ...int) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	code := http.StatusOK
	if len(codes) > 0 {
		code = codes[0]
	}

	// This must come before WriteHeader, otherwise the header will not be set correctly.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write(b)
	return err
}

func OK(w http.ResponseWriter, data any, codes ...int) error {
	code := http.StatusOK
	if len(codes) > 0 {
		code = codes[0]
	}

	return JSON(w, &Body{
		Code: code,
		Data: data,
	}, code)
}

func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	body := NewJSONError(err)
	if err := JSON(w, body, body.Code); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
