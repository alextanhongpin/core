package response

import (
	"errors"
	"net/http"

	"github.com/alextanhongpin/errors/causes"
	"github.com/alextanhongpin/errors/codes"
)

type PageInfo struct {
	HasPrevPage bool   `json:"hasPrevPage"`
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor"`
	EndCursor   string `json:"endCursor"`
}

type Body struct {
	Data     any        `json:"data,omitempty"`
	Error    *JSONError `json:"error,omitempty"`
	PageInfo *PageInfo  `json:"pageInfo,omitempty"`
}

type JSONError struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Errors  ValidationErrors `json:"errors,omitempty"`
}

func BodyError(err error) (*Body, int) {
	var ve ValidationErrors
	if errors.As(err, &ve) {
		code := http.StatusBadRequest
		return &Body{
			Error: &JSONError{
				Code:    http.StatusText(code),
				Message: err.Error(),
				Errors:  ve,
			},
		}, code
	}

	var det causes.Detail
	if errors.As(err, &det) {
		code := codes.HTTP(det.Code())
		return &Body{
			Error: &JSONError{
				Code:    det.Kind(),
				Message: det.Message(),
			},
		}, code
	}

	code := http.StatusInternalServerError
	return &Body{
		Error: &JSONError{
			Code:    http.StatusText(code),
			Message: "An unexpected error has occured. Please try again later",
		},
	}, code
}
