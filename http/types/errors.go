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

	ErrInternal     = errors.Get("api.internal")
	ErrBadRequest   = errors.Get("api.bad_request")
	ErrUnauthorized = errors.Get("api.unauthorized")
)

var errorKindToStatusCode = map[errors.Kind]int{
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

func ErrorStatusCode(kind errors.Kind) int {
	statusCode, ok := errorKindToStatusCode[kind]
	if !ok {
		return http.StatusInternalServerError
	}

	return statusCode
}
