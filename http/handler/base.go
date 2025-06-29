package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alextanhongpin/core/http/request"
	"github.com/alextanhongpin/core/http/response"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
)

type BaseHandler struct {
	logger *slog.Logger
}

func (h BaseHandler) WithLogger(logger *slog.Logger) BaseHandler {
	h.logger = logger
	return h
}

func (h BaseHandler) ReadJSON(r *http.Request, req any) error {
	return request.DecodeJSON(r, req)
}

func (h BaseHandler) OK(w http.ResponseWriter, data any, codes ...int) {
	response.OK(w, data, codes...)
}

func (h BaseHandler) JSON(w http.ResponseWriter, data any, codes ...int) {
	response.JSON(w, data, codes...)
}

func (h BaseHandler) ErrorJSON(w http.ResponseWriter, err error) {
	response.ErrorJSON(w, err)
}

func (h BaseHandler) NoContent(w http.ResponseWriter) {
	response.NoContent(w)
}

func (h BaseHandler) Next(w http.ResponseWriter, r *http.Request, err error) {
	if h.logger != nil {
		attrs := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("pattern", r.Pattern),
		}

		var ve interface {
			Map() map[string]any
		}
		var c *cause.Error

		switch {
		case errors.As(err, &ve):
			// If the error is a validation error, we log it as a warning.
			h.logger.WarnContext(r.Context(), "validation error occurred",
				append(attrs,
					slog.Any("errors", ve.Map()),
					slog.Int("code", http.StatusBadRequest),
				)...,
			)
		case errors.As(err, &c):
			h.logger.ErrorContext(r.Context(), "error occurred",
				append(attrs,
					slog.Any("err", c),
					slog.Int("code", codes.HTTP(c.Code)),
				)...,
			)
		default:
			// For any other error, we log it as an error.
			h.logger.ErrorContext(r.Context(), "internal error occurred",
				append(attrs,
					slog.Any("err", err.Error()),
					slog.Int("code", http.StatusInternalServerError),
				)...,
			)
		}
	}

	h.ErrorJSON(w, err)
}
