// Package handlers provides a lightweight, testable HTTP-like request/response
// pattern for internal service communication, message processing, and testing.
// It abstracts the HTTP layer to enable easier unit testing and decoupled
// service architectures.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	// ErrPatternNotFound is returned when a requested pattern/route is not registered
	ErrPatternNotFound = errors.New("handlers: pattern not found")
	// ErrRequestTimeout is returned when a request exceeds its timeout
	ErrRequestTimeout = errors.New("handlers: request timeout")
)

// Request represents a request with pattern, body, metadata, and context.
// It abstracts HTTP requests for internal use.
type Request struct {
	Pattern   string            `json:"pattern"`
	Body      io.Reader         `json:"-"`
	Meta      map[string]string `json:"meta"`
	Timestamp time.Time         `json:"timestamp"`
	ctx       context.Context
}

// NewRequest creates a new request with the specified pattern and body.
func NewRequest(pattern string, body io.Reader) *Request {
	return &Request{
		Pattern:   pattern,
		Body:      body,
		Meta:      make(map[string]string),
		Timestamp: time.Now(),
		ctx:       context.Background(),
	}
}

// WithContext sets the context for the request.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Context returns the request's context.
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// WithMeta adds metadata to the request.
func (r *Request) WithMeta(key, value string) *Request {
	r.Meta[key] = value
	return r
}

// GetMeta retrieves metadata from the request.
func (r *Request) GetMeta(key string) (string, bool) {
	value, ok := r.Meta[key]
	return value, ok
}

// Decode unmarshals the request body into the provided value.
func (r *Request) Decode(v any) error {
	if r.Body == nil {
		return fmt.Errorf("handlers: request body is nil")
	}
	return json.NewDecoder(r.Body).Decode(v)
}

// Clone creates a copy of the request with a new body.
func (r *Request) Clone(newBody io.Reader) *Request {
	meta := make(map[string]string)
	for k, v := range r.Meta {
		meta[k] = v
	}

	return &Request{
		Pattern:   r.Pattern,
		Body:      newBody,
		Meta:      meta,
		Timestamp: r.Timestamp,
		ctx:       r.ctx,
	}
}

// Response represents a response with body, status, headers, and metadata.
type Response struct {
	Body      *bytes.Buffer     `json:"-"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Meta      map[string]string `json:"meta"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
}

// NewResponse creates a new response.
func NewResponse() *Response {
	return &Response{
		Body:      bytes.NewBuffer(nil),
		Status:    200, // Default to OK
		Headers:   make(map[string]string),
		Meta:      make(map[string]string),
		Timestamp: time.Now(),
	}
}

// WriteStatus sets the response status code.
func (r *Response) WriteStatus(status int) {
	r.Status = status
}

// SetHeader sets a response header.
func (r *Response) SetHeader(key, value string) {
	r.Headers[key] = value
}

// GetHeader retrieves a response header.
func (r *Response) GetHeader(key string) (string, bool) {
	value, ok := r.Headers[key]
	return value, ok
}

// SetMeta sets response metadata.
func (r *Response) SetMeta(key, value string) {
	r.Meta[key] = value
}

// GetMeta retrieves response metadata.
func (r *Response) GetMeta(key string) (string, bool) {
	value, ok := r.Meta[key]
	return value, ok
}

// Write writes data to the response body.
func (r *Response) Write(b []byte) (int, error) {
	return r.Body.Write(b)
}

// Encode marshals a value to JSON and writes it to the response body.
func (r *Response) Encode(v any) error {
	r.SetHeader("Content-Type", "application/json")
	return json.NewEncoder(r.Body).Encode(v)
}

// Decode unmarshals the response body into the provided value.
func (r *Response) Decode(v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// String returns the response body as a string.
func (r *Response) String() string {
	return r.Body.String()
}

// Bytes returns the response body as bytes.
func (r *Response) Bytes() []byte {
	return r.Body.Bytes()
}

// ResponseWriter interface for writing responses.
type ResponseWriter interface {
	io.Writer
	WriteStatus(status int)
	SetHeader(key, value string)
	SetMeta(key, value string)
	Encode(v any) error
}

// HandlerFunc is a function type that implements Handler.
type HandlerFunc func(w ResponseWriter, r *Request) error

// Handle implements the Handler interface.
func (fn HandlerFunc) Handle(w ResponseWriter, r *Request) error {
	return fn(w, r)
}

// Handler interface for processing requests.
type Handler interface {
	Handle(w ResponseWriter, r *Request) error
}

// Middleware function type for wrapping handlers.
type Middleware func(Handler) Handler

// Router manages request routing and middleware.
type Router struct {
	routes     map[string]Handler
	middleware []Middleware
	timeout    time.Duration
}

// NewRouter creates a new router.
func NewRouter() *Router {
	return &Router{
		routes:  make(map[string]Handler),
		timeout: 30 * time.Second, // Default timeout
	}
}

// WithTimeout sets the default timeout for requests.
func (r *Router) WithTimeout(timeout time.Duration) *Router {
	r.timeout = timeout
	return r
}

// Use adds middleware to the router.
func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

// Handle registers a handler for a pattern.
func (r *Router) Handle(pattern string, handler Handler) {
	r.routes[pattern] = r.wrapMiddleware(handler)
}

// HandleFunc registers a handler function for a pattern.
func (r *Router) HandleFunc(pattern string, fn func(w ResponseWriter, r *Request) error) {
	r.Handle(pattern, HandlerFunc(fn))
}

// Handler retrieves a handler for a pattern.
func (r *Router) Handler(pattern string) (Handler, bool) {
	h, ok := r.routes[pattern]
	return h, ok
}

// Do processes a request and returns a response.
func (r *Router) Do(req *Request) (*Response, error) {
	h, ok := r.Handler(req.Pattern)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPatternNotFound, req.Pattern)
	}

	// Create response
	res := NewResponse()
	startTime := time.Now()

	// Apply timeout if context doesn't already have one
	ctx := req.Context()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Execute handler with timeout
	done := make(chan error, 1)
	go func() {
		done <- h.Handle(res, req)
	}()

	select {
	case err := <-done:
		res.Duration = time.Since(startTime)
		if err != nil {
			return res, err
		}
		return res, nil
	case <-ctx.Done():
		res.Duration = time.Since(startTime)
		return res, ErrRequestTimeout
	}
}

// wrapMiddleware applies all middleware to a handler.
func (r *Router) wrapMiddleware(h Handler) Handler {
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	return h
}

// Common middleware implementations

// LoggingMiddleware logs requests and responses.
func LoggingMiddleware(logger func(pattern string, duration time.Duration, status int)) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) error {
			start := time.Now()
			err := next.Handle(w, r)

			duration := time.Since(start)
			status := 200
			if resp, ok := w.(*Response); ok {
				status = resp.Status
			}

			logger(r.Pattern, duration, status)
			return err
		})
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error.
func RecoveryMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) error {
			defer func() {
				if rec := recover(); rec != nil {
					w.WriteStatus(500)
					w.Encode(map[string]interface{}{
						"error": "internal server error",
						"panic": fmt.Sprintf("%v", rec),
					})
				}
			}()
			return next.Handle(w, r)
		})
	}
}

// AuthMiddleware validates authentication.
func AuthMiddleware(validateToken func(token string) error) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) error {
			token, ok := r.GetMeta("Authorization")
			if !ok {
				w.WriteStatus(401)
				return w.Encode(map[string]string{"error": "unauthorized"})
			}

			if err := validateToken(token); err != nil {
				w.WriteStatus(401)
				return w.Encode(map[string]string{"error": "invalid token"})
			}

			return next.Handle(w, r)
		})
	}
}

// RateLimitMiddleware implements basic rate limiting.
func RateLimitMiddleware(limiter func(pattern string) bool) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) error {
			if !limiter(r.Pattern) {
				w.WriteStatus(429)
				return w.Encode(map[string]string{"error": "rate limit exceeded"})
			}
			return next.Handle(w, r)
		})
	}
}
