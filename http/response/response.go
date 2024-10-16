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
	Code     int       `json:"-"`
	Data     any       `json:"data,omitempty"`
	Error    *Error    `json:"error,omitempty"`
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

func NewData(data any) *Body {
	return &Body{
		Code: http.StatusOK,
		Data: data,
	}
}

func NewBody(data any, err error) *Body {
	if err != nil {
		return NewError(err)
	}

	return &Body{
		Code: http.StatusOK,
		Data: data,
	}
}

type Error struct {
	Code             string           `json:"code"`
	Message          string           `json:"message"`
	ValidationErrors ValidationErrors `json:"validationErrors,omitempty"`
}

func NewError(err error) *Body {
	var ve ValidationErrors
	if errors.As(err, &ve) {
		code := http.StatusBadRequest

		return &Body{
			Code: code,
			Error: &Error{
				Code:             http.StatusText(code),
				Message:          err.Error(),
				ValidationErrors: ve,
			},
		}
	}

	var det causes.Detail
	if errors.As(err, &det) {
		code := codes.HTTP(det.Code())

		return &Body{
			Code: code,
			Error: &Error{
				Code:    det.Kind(),
				Message: det.Message(),
			},
		}
	}

	code := http.StatusInternalServerError

	return &Body{
		Code: code,
		Error: &Error{
			Code:    http.StatusText(code),
			Message: "An unexpected error has occured. Please try again later",
		},
	}
}
