package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alextanhongpin/errors/causes"
	"github.com/alextanhongpin/errors/codes"
)

var (
	ErrBadRequest         = causes.New(codes.BadRequest, "bad_request", "The input you provided is invalid")
	ErrConflict           = causes.New(codes.Conflict, "conflict", "The action may have conflict")
	ErrExists             = causes.New(codes.Exists, "exists", "There may be duplicate entries")
	ErrForbidden          = causes.New(codes.Forbidden, "forbidden", "You do not have permission to perform this action")
	ErrInternal           = causes.New(codes.Internal, "internal_server_error", "Oops, please try again later")
	ErrNotFound           = causes.New(codes.NotFound, "not_found", "The thing you are looking for does not exist or may have been deleted")
	ErrPreconditionFailed = causes.New(codes.PreconditionFailed, "precondition_failed", "The action cannot be completed")
	ErrUnauthorized       = causes.New(codes.Unauthorized, "unauthorized", "You are not logged in")
	ErrUnknown            = causes.New(codes.Unknown, "unknown", "An error has occured")
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Payload[T any] struct {
	Data  T      `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Links *Links `json:"links,omitempty"`
}

func (p *Payload[T]) WithLinks(links *Links) *Payload[T] {
	p.Links = links
	return p
}

func (p *Payload[T]) WithMeta(meta *Meta) *Payload[T] {
	p.Meta = meta
	return p
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
	var c causes.Detail
	if !errors.As(err, &c) {
		JSONError(w, ErrInternal)
		return
	}

	d := c.Detail()

	res := &Payload[any]{
		Error: &Error{
			Code:    d.Kind(),
			Message: d.Message(),
		},
	}

	JSON(w, res, codes.HTTP(d.Code()))
}

func OK[T any](t T) *Payload[T] {
	return &Payload[T]{
		Data: t,
	}
}
