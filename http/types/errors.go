package types

import (
	"net/http"

	"github.com/alextanhongpin/go-core-microservice/types/errors"
)

var statusCodeByErrorKind = map[errors.Kind]int{
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
