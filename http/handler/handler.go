package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/http/request"
	"github.com/alextanhongpin/core/http/response"
)

type ServiceFunc[K, V any] func(ctx context.Context, req K) (V, error)

func (f ServiceFunc[K, V]) Run(ctx context.Context, req K) (V, error) {
	return f(ctx, req)
}

type Service[K, V any] interface {
	Run(ctx context.Context, req K) (V, error)
}

func NewFunc[K, V any](service ServiceFunc[K, V]) *Handler[K, V] {
	return New(service)
}

func New[K, V any](service Service[K, V]) *Handler[K, V] {
	return &Handler[K, V]{
		BaseHandler: &baseHandler{},
		Service:     service,
	}
}

type Handler[K, V any] struct {
	BaseHandler
	Service Service[K, V]
}

func (h *Handler[K, V]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req K
	if err := h.Decode(r, &req); err != nil {
		h.EncodeError(w, err)

		return
	}

	resp, err := h.Service.Run(r.Context(), req)
	if err != nil {
		h.EncodeError(w, err)

		return
	}

	h.Encode(w, resp)
}

type BaseHandler interface {
	Decode(r *http.Request, req any) error
	Encode(w http.ResponseWriter, data any)
	EncodeError(w http.ResponseWriter, err error)
}

type baseHandler struct{}

func (b *baseHandler) Decode(r *http.Request, req any) error {
	return request.DecodeJSON(r, req)
}

func (b *baseHandler) Encode(w http.ResponseWriter, data any) {
	response.JSON(w, data)
}

func (b *baseHandler) EncodeError(w http.ResponseWriter, err error) {
	response.ErrorJSON(w, err)
}
