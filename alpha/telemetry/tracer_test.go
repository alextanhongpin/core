package telemetry_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/alpha/telemetry"
)

func TestTracer(t *testing.T) {
	svc := &testServiceWrapper{new(testService)}
	ctx := telemetry.NewTracerContext(context.Background())
	got := svc.Greet(ctx, "John")
	want := "Hello, John"
	if want != got {
		t.Fatalf("want %s, got %s", want, got)
	}

	tracer, ok := telemetry.TracerFromContext(ctx)
	if !ok {
		t.Fatal("missing tracer")
	}

	b, err := json.MarshalIndent(tracer, "", " ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))
}

type testService struct{}

func (svc *testService) Greet(ctx context.Context, name string) string {
	return "Hello, " + name
}

type testServiceWrapper struct {
	*testService
}

func (svc *testServiceWrapper) Greet(ctx context.Context, name string) (msg string) {
	defer func(name string) {
		_ = telemetry.Trace(ctx, telemetry.Step{
			Name:        "Greet",
			Description: "Greet someone",
			Request: map[string]any{
				"name": name,
			},
			Response: map[string]any{
				"msg": msg,
			},
		})
	}(name)
	msg = svc.testService.Greet(ctx, name)

	return
}
