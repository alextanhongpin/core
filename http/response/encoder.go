package response

import (
	"cmp"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alextanhongpin/errors/causes"
	codec "github.com/alextanhongpin/errors/codes"
)

type JSONEncoder struct {
	w    http.ResponseWriter
	r    *http.Request
	body *Body
	err  error
	code int
}

func NewJSONEncoder(w http.ResponseWriter, r *http.Request) *JSONEncoder {
	return &JSONEncoder{
		w: w,
		r: r,
	}
}

func (e *JSONEncoder) SetData(data any, codes ...int) {
	e.body = &Body{
		Data: data,
	}
	e.code = cmp.Or(head(codes), http.StatusOK)
}

func (e *JSONEncoder) SetError(err error) {
	e.err = err
	e.code = http.StatusInternalServerError

	var ve ValidationErrors
	if errors.As(err, &ve) {
		e.code = http.StatusBadRequest
		e.body = &Body{
			Error: &Error{
				Code:             http.StatusText(e.code),
				Message:          err.Error(),
				ValidationErrors: ve,
			},
		}

		return
	}

	var det causes.Detail
	if errors.As(err, &det) {
		e.code = codec.HTTP(det.Code())
		e.body = &Body{
			Error: &Error{
				Code:    det.Kind(),
				Message: det.Message(),
			},
		}

		return
	}

	e.body = &Body{
		Error: &Error{
			Code:    http.StatusText(http.StatusInternalServerError),
			Message: "An unexpected error occurred",
		},
	}
}

func (e *JSONEncoder) Pipe(v any, err error) {
	if err != nil {
		e.SetError(err)
	} else {
		e.SetData(v)
	}
}

func (e *JSONEncoder) Flush() {
	w := e.w
	if e.body == nil && e.err == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.code)
	if err := json.NewEncoder(w).Encode(e.body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func head[T any](vs []T) (v T) {
	if len(vs) > 0 {
		return vs[0]
	}

	return
}
