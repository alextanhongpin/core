package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewTracer(t *testing.T) {
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	shutdown, err := NewTracer(t.Context(), TracerConfig{ServiceName: "test", SampleRate: 1})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoError(t, shutdown(ctx))
}

func TestHTTPHandlerSpanNames(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
	srv := HTTPHandler(mux)

	for _, path := range []string{"/users/1", "/missing", "/health"} {
		srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	var names []string
	for _, s := range sr.Ended() {
		names = append(names, s.Name())
	}
	assert.Equal(t, []string{"GET /users/{id}", "GET"}, names)
}

func TestNewTransportKeepsDefaults(t *testing.T) {
	tr := newTransport()
	assert.NotNil(t, tr.Proxy, "proxy from environment")
	assert.NotNil(t, tr.DialContext)
	assert.NotZero(t, tr.TLSHandshakeTimeout)
	assert.True(t, tr.ForceAttemptHTTP2)
	assert.Equal(t, 10, tr.MaxIdleConnsPerHost)
}
