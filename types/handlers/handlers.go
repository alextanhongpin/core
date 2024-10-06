package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrPatternNotFound = errors.New("pattern not found")

type Request struct {
	Pattern string
	Body    io.Reader
	Meta    map[string]string
	ctx     context.Context
}

func NewRequest(pattern string, body io.Reader) *Request {
	return &Request{
		Pattern: pattern,
		Body:    body,
		Meta:    make(map[string]string),
		ctx:     context.Background(),
	}
}

func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

func (r *Request) Context() context.Context {
	return r.ctx
}

func (r *Request) Decode(v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type Response struct {
	Body   *bytes.Buffer
	Status int
}

func NewResponse() *Response {
	return &Response{
		Body: bytes.NewBuffer(nil),
	}
}

func (r *Response) WriteStatus(status int) {
	r.Status = status
}

func (r *Response) Write(b []byte) (int, error) {
	return r.Body.Write(b)
}

func (r *Response) Encode(v any) error {
	return json.NewEncoder(r.Body).Encode(v)
}

func (r *Response) Decode(v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type ResponseWriter interface {
	io.Writer
	WriteStatus(status int)
	Encode(v any) error
}

type HandlerFunc func(w ResponseWriter, r *Request) error

func (fn HandlerFunc) Handle(w ResponseWriter, r *Request) error {
	return fn(w, r)
}

type Handler interface {
	Handle(w ResponseWriter, r *Request) error
}

type Router struct {
	routes map[string]Handler
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]Handler),
	}
}

func (r *Router) Handle(pattern string, handler Handler) {
	r.routes[pattern] = handler
}

func (r *Router) HandleFunc(pattern string, fn func(w ResponseWriter, r *Request) error) {
	r.routes[pattern] = HandlerFunc(fn)
}

func (r *Router) Handler(pattern string) (Handler, bool) {
	h, ok := r.routes[pattern]
	return h, ok
}

func (r *Router) Do(req *Request) (*Response, error) {
	h, ok := r.Handler(req.Pattern)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPatternNotFound, req.Pattern)
	}
	res := NewResponse()
	if err := h.Handle(res, req); err != nil {
		return nil, err
	}
	return res, nil
}
