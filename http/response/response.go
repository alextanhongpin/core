package response

import (
	"errors"
	"net/http"

	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
)

const message = "An unexpected error occurred. Please try again later."

type PageInfo struct {
	HasPrevPage bool   `json:"hasPrevPage"`
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor"`
	EndCursor   string `json:"endCursor"`
}

type Body struct {
	Data     any       `json:"data,omitempty"`
	Error    *Error    `json:"error,omitempty"`
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Errors  map[string]any `json:"errors,omitempty"`
}

func BodyError(err error) (*Body, int) {
	var ve interface {
		Map() map[string]any
	}
	if errors.As(err, &ve) {
		code := http.StatusBadRequest
		return &Body{
			Error: &Error{
				Code:    http.StatusText(code),
				Message: err.Error(),
				Errors:  ve.Map(),
			},
		}, code
	}

	var c *cause.Error
	if errors.As(err, &c) {
		code := codes.HTTP(c.Code)
		return &Body{
			Error: &Error{
				Code:    c.Name,
				Message: c.Message,
			},
		}, code
	}

	code := http.StatusInternalServerError
	return &Body{
		Error: &Error{
			Code:    http.StatusText(code),
			Message: message,
		},
	}, code
}
