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

// DecodeJSON decodes the json to struct and performs
// validation.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	// Duplicate the request to a buffer.
	var t T
	var buf bytes.Buffer
	rr := io.TeeReader(r.Body, &buf)

	if err := json.NewDecoder(rr).Decode(&t); err != nil && !errors.Is(err, io.EOF) {
		// Set back to the body as if it was never read before.
		// This allows us to log the request body.
		r.Body = io.NopCloser(&buf)

		return t, err
	}

	r.Body = io.NopCloser(&buf)

	if err := validate.Struct(&t); err != nil {
		return t, err
	}

	return t, nil
}

// EncodeJSON encodes the result to json representation.
func EncodeJSON[T any](w http.ResponseWriter, statusCode int, res T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// EncodeError encodes the error as json response. Status code is inferred from
// the error kind.
func EncodeJSONError(w http.ResponseWriter, err error) {
	appErr, statusCode := errorToAppError(err)
	result := types.Result[any]{
		Error: &types.Error{
			Code:    appErr.Code,
			Message: appErr.Error(),
		},
	}

	EncodeJSON(w, statusCode, result)
}

func errorToAppError(err error) (*errors.Error, int) {
	var cause *errors.Error
	if errors.As(err, &cause) {
		return cause, types.ErrorStatusCode(errors.Kind(cause.Kind))
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		apiErr := types.ErrBadRequest.Copy()
		apiErr.Message = err.Error()

		return apiErr, http.StatusBadRequest
	}

	// Avoid exposing internal error.
	return types.ErrInternal, http.StatusInternalServerError
}
