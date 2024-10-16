package handlers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/handlers"
	"github.com/stretchr/testify/assert"
)

type MessageRequest struct {
	Name string `json:"name"`
}

type MessageResponse struct {
	Msg string `json:"msg"`
}

func TestHandlers(t *testing.T) {
	r := handlers.NewRouter()
	r.HandleFunc("test", func(w handlers.ResponseWriter, r *handlers.Request) error {
		var req MessageRequest
		if err := r.Decode(&req); err != nil {
			return err
		}
		fmt.Println("hitting")
		w.WriteStatus(123)
		return w.Encode(&MessageResponse{
			Msg: "hello world",
		})
	})

	req := handlers.NewRequest("test", strings.NewReader(`{"name": "John"}`))
	resp, err := r.Do(req)

	is := assert.New(t)
	is.Nil(err)
	is.Equal(123, resp.Status)

	var res MessageResponse
	is.Nil(resp.Decode(&res))
	is.Equal("hello world", res.Msg)
}

func TestHandlers_NotFound(t *testing.T) {
	r := handlers.NewRouter()
	req := handlers.NewRequest("test", strings.NewReader(`{"name": "John"}`))
	resp, err := r.Do(req)
	is := assert.New(t)
	is.ErrorIs(err, handlers.ErrPatternNotFound)
	is.Nil(resp)
}