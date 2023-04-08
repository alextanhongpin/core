package encoding

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/alextanhongpin/go-core-microservice/http/types"
	"github.com/alextanhongpin/go-core-microservice/types/errors"
	validator "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Decode decodes and validates the body.
func Decode[T any](w http.ResponseWriter, r *http.Request) (t T, err error) {
	// Duplicate the request to a buffer.
	var buf bytes.Buffer
	rr := io.TeeReader(r.Body, &buf)

	if err = json.NewDecoder(rr).Decode(&t); err != nil && !errors.Is(err, io.EOF) {
		// Set back to the body as if it was never read before.
		// This allows us to log the request body.
		r.Body = io.NopCloser(&buf)

		return
	}

	r.Body = io.NopCloser(&buf)

	if err = validate.Struct(&t); err != nil {
		return
	}

	return t, nil
}

// EncodeError encodes the error as json response.
// Status code is inferred from the error kind.
func EncodeError(w http.ResponseWriter, err error) {
	appErr, statusCode := errorToAppError(err)
	result := types.Error{
		Code:    appErr.Code,
		Message: appErr.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// EncodeResult only encodes dto as json response.
func EncodeResult[T any](w http.ResponseWriter, statusCode int, res T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func errorToAppError(err error) (*errors.Error, int) {
	var cause *errors.Error
	var validationErrors validator.ValidationErrors

	switch {
	case errors.As(err, &cause):
		statusCode, found := types.ErrorKindToStatusCode[errors.Kind(cause.Kind)]
		if !found {
			statusCode = http.StatusInternalServerError
		}

		return cause, statusCode
	case errors.As(err, &validationErrors):
		apiErr := types.ErrBadRequest.Copy()
		apiErr.Message = err.Error()

		return apiErr, http.StatusBadRequest
	default:
		// Avoid exposing internal error.
		return types.ErrInternal, http.StatusInternalServerError
	}
}
