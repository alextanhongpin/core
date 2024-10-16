package response

import (
	"cmp"
	"encoding/json"
	"net/http"
)

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func EncodeJSON(w http.ResponseWriter, body any, code int) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	// This must come before WriteHeader, otherwise the header will not be set correctly.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(cmp.Or(code, http.StatusOK))
	_, err = w.Write(b)
	return err
}

func EncodeBody(w http.ResponseWriter, body *Body) error {
	return EncodeJSON(w, body, body.Code)
}

func EncodeData(w http.ResponseWriter, data any, code int) error {
	body := NewData(data, code)

	return EncodeJSON(w, body, body.Code)
}

func EncodeError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	body := NewError(err)

	if err := EncodeJSON(w, body, body.Code); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
