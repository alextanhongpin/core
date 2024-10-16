package response

import (
	"cmp"
	"encoding/json"
	"log/slog"
	"net/http"
)

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

type JSONEncoder struct {
	w      http.ResponseWriter
	Logger *slog.Logger
}

func NewJSONEncoder(w http.ResponseWriter) *JSONEncoder {
	return &JSONEncoder{
		w:      w,
		Logger: slog.Default(),
	}
}

func (enc *JSONEncoder) Encode(body any, code int) {
	// This must come before WriteHeader, otherwise the header will not be set correctly.
	w := enc.w
	logger := enc.Logger
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(body)
	if err != nil {
		logger.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(cmp.Or(code, http.StatusOK))
	_, err = w.Write(b)
	if err != nil {
		logger.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (enc *JSONEncoder) Body(body *Body) {
	enc.Encode(body, body.Code)
}

func (enc *JSONEncoder) Data(data any, code int) {
	body := NewData(data)
	body.Code = code
	enc.Body(body)
}

func (enc *JSONEncoder) Error(err error) {
	body := NewError(err)
	enc.Encode(body, body.Code)
}
