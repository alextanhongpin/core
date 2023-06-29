package telemetry

import (
	"context"
	"encoding/json"
	"io"
)

type Error struct {
	error
}

func (e *Error) MarshalJSON() ([]byte, error) {
	if e.error != nil {
		return json.Marshal(e.error.Error())
	}

	return nil, nil
}

type trackerKey string

var tracerCtxKey trackerKey = "tracer_ctx"

type Step struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Request     any    `json:"request"`
	Response    any    `json:"response"`
	Error       *Error `json:"error"`
}

type Tracer struct {
	Steps []Step `json:"steps"`
}

func NewTracerContext(ctx context.Context) context.Context {
	if v := ctx.Value(tracerCtxKey); v != nil {
		return ctx
	}

	return context.WithValue(ctx, tracerCtxKey, new(Tracer))
}

func Trace(ctx context.Context, step Step) bool {
	tracer, ok := TracerFromContext(ctx)
	if !ok {
		return false
	}

	tracer.Add(step)

	return true
}

func TracerFromContext(ctx context.Context) (*Tracer, bool) {
	v, ok := ctx.Value(tracerCtxKey).(*Tracer)
	return v, ok
}

func (t *Tracer) Add(steps ...Step) {
	t.Steps = append(t.Steps, steps...)
}

func (t *Tracer) Write(w io.Writer) error {
	b, err := json.MarshalIndent(t.Steps, "", " ")
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}
