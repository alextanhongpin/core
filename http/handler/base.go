// Package handler provides a base controller for HTTP handlers with common
// functionality including JSON handling, error management, and structured logging.
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

// BaseHandler provides common HTTP handler functionality including JSON processing,
// error handling, response formatting, and structured logging.
//
// It serves as a foundation for building REST API controllers with consistent
// error handling, logging, and response patterns.
//
// Example usage:
//
//	type UserController struct {
//		handler.BaseHandler
//		userService *UserService
//	}
//
//	func NewUserController(userService *UserService) *UserController {
//		return &UserController{
//			BaseHandler: handler.BaseHandler{}.WithLogger(slog.Default()),
//			userService: userService,
//		}
//	}
//
//	func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
//		var req CreateUserRequest
//		if err := c.ReadJSON(r, &req); err != nil {
//			c.Next(w, r, err)
//			return
//		}
//
//		user, err := c.userService.Create(r.Context(), req)
//		if err != nil {
//			c.Next(w, r, err)
//			return
//		}
//
//		c.JSON(w, user, http.StatusCreated)
//	}
type BaseHandler struct {
	logger *slog.Logger
}

// WithLogger returns a new BaseHandler with the provided logger.
// This enables structured logging for all handler operations.
//
// Example:
//
//	handler := BaseHandler{}.WithLogger(slog.Default())
func (h BaseHandler) WithLogger(logger *slog.Logger) BaseHandler {
	h.logger = logger
	return h
}

func (h BaseHandler) Logger() *slog.Logger {
	return h.logger
}

// ReadJSON decodes JSON from the request body into the provided struct.
// It handles content-type validation and JSON parsing errors automatically.
//
// The req parameter should be a pointer to a struct that implements
// validation if needed (e.g., has a Validate() error method).
//
// Example:
//
//	var req CreateUserRequest
//	if err := c.ReadJSON(r, &req); err != nil {
//		c.Next(w, r, err)
//		return
//	}
func (h BaseHandler) ReadJSON(r *http.Request, req any) error {
	return request.DecodeJSON(r, req)
}

// JSON writes a JSON response with status code.
// This is an alias for OK method for better semantic clarity.
//
// Example:
//
//	c.JSON(w, users, 200)

// ErrorJSON writes an error response in JSON format.
// It automatically determines the appropriate HTTP status code based on the error type.
//
// Supports:
// - Validation errors (400 Bad Request)
// - Cause errors with specific codes
// - Generic errors (500 Internal Server Error)
//
// Example:
//
//	c.ErrorJSON(w, cause.New(codes.NotFound, "user/not_found", "User not found"))
//
// ErrorJSON is a helper method to standardize error responses
func (h BaseHandler) ErrorJSON(w http.ResponseWriter, err error) {
	response.ErrorJSON(w, err)
}

// NoContent writes a 204 No Content response.
// Typically used for successful DELETE operations or updates with no response body.
//
// Example:
//
//	c.NoContent(w) // 204 No Content
func (h BaseHandler) NoContent(w http.ResponseWriter) {
	response.NoContent(w)
}

// Next handles error processing and logging, then writes an appropriate error response.
// This is the central error handling method that should be called whenever an error occurs.
//
// It provides intelligent error categorization and logging:
// - Validation errors: logged as warnings with 400 status
// - Cause errors: logged as errors with mapped HTTP status codes
// - Generic errors: logged as errors with 500 status
//
// The method automatically logs relevant request context including method, path, and pattern.
//
// Example:
//
//	if err := c.userService.GetUser(ctx, id); err != nil {
//		c.Next(w, r, err)
//		return
//	}
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

// GetRequestID extracts the request ID from the request context.
// This is useful for correlation logging and tracing.
//
// Example:
//
//	if requestID := c.GetRequestID(r); requestID != "" {
//		// Use request ID for logging or response headers
//	}
func (h BaseHandler) GetRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	if id := r.Header.Get("X-Correlation-ID"); id != "" {
		return id
	}
	return ""
}

// SetRequestID sets the request ID in the response headers for client correlation.
//
// Example:
//
//	c.SetRequestID(w, requestID)
func (h BaseHandler) SetRequestID(w http.ResponseWriter, requestID string) {
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}
}

// JSON writes a JSON response with status code.
// This is an alias for OK method for better semantic clarity.
//
// Example:
//
//	c.JSON(w, users, 200)
func (h BaseHandler) JSON(w http.ResponseWriter, data any, code int) {
	response.JSON(w, data, code)
}

// Body writes a JSON response with a structured body.
// It wraps the data in a response.Body struct, which includes a Data field.
// This is useful for consistent API responses that include metadata or additional fields.
func (h BaseHandler) Body(w http.ResponseWriter, data any, code int) {
	response.JSON(w, &response.Body{Data: data}, code)
}
