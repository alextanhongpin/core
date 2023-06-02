package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alextanhongpin/errcodes"
)

var (
	ErrBadRequest         = errcodes.New(errcodes.BadRequest, "bad_request", "The input you provided is invalid")
	ErrConflict           = errcodes.New(errcodes.Conflict, "conflict", "The action may have conflict")
	ErrExists             = errcodes.New(errcodes.Exists, "exists", "There may be duplicate entries")
	ErrForbidden          = errcodes.New(errcodes.Forbidden, "forbidden", "You do not have permission to perform this action")
	ErrInternal           = errcodes.New(errcodes.Internal, "internal_server_error", "Oops, please try again later")
	ErrNotFound           = errcodes.New(errcodes.NotFound, "not_found", "The thing you are looking for does not exist or may have been deleted")
	ErrPreconditionFailed = errcodes.New(errcodes.PreconditionFailed, "precondition_failed", "The action cannot be completed")
	ErrUnauthorized       = errcodes.New(errcodes.Unauthorized, "unauthorized", "You are not logged in")
	ErrUnknown            = errcodes.New(errcodes.Unknown, "unknown", "An error has occured")
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Payload[T any] struct {
	Data  *T     `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Links *Links `json:"links,omitempty"`
}

type Meta map[string]any

type Links struct {
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
}

// JSON encodes the result to json representation.
func JSON[T any](w http.ResponseWriter, res T, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// JSONError encodes the error as json response. Status code is inferred from
// the error kind.
func JSONError(w http.ResponseWriter, err error) {
	var errCode *errcodes.Error
	if !errors.As(err, &errCode) {
		errCode = ErrInternal
	}

	res := &Payload[any]{
		Error: &Error{
			Code:    string(errCode.Code),
			Message: errCode.Message,
		},
	}

	JSON(w, res, errcodes.HTTPStatusCode(errCode.Kind))
}

func OK[T any](t *T) *Payload[T] {
	return &Payload[T]{
		Data: t,
	}
}
