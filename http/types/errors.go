package types

import (
	"net/http"

	_ "embed"

	"github.com/BurntSushi/toml"
	"github.com/alextanhongpin/go-core-microservice/types/errors"
)

var (
	//go:embed errors.toml
	errorBytes []byte
	_          = errors.MustLoad(errorBytes, toml.Unmarshal)

	ErrInternal   = errors.Get("api.internal")
	ErrBadRequest = errors.Get("api.bad_request")
)

var ErrorKindToStatusCode = map[errors.Kind]int{
	errors.AlreadyExists: http.StatusConflict,
	errors.BadInput:      http.StatusBadRequest,
	errors.Conflict:      http.StatusConflict,
	errors.Forbidden:     http.StatusForbidden,
	errors.Internal:      http.StatusInternalServerError,
	errors.NotFound:      http.StatusNotFound,
	errors.Unauthorized:  http.StatusUnauthorized,
	errors.Unknown:       http.StatusInternalServerError,
	errors.Unprocessable: http.StatusUnprocessableEntity,
}
