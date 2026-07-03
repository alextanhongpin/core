package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	SampleRate     float64
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: otelhttp.NewTransport(
			&http.Transport{
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}),
		),
	}
}

func HTTPHandler(next http.Handler) http.Handler {
	// Add HTTP instrumentation for the whole server.
	return otelhttp.NewHandler(
		next,
		"http-server",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.Pattern)
		}),
		otelhttp.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/health"
		}),
	)
}

func NewTracer(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("merging resource: %w", err)
	}

	// https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/
	// TODO: Setup OTEL_EXPORTER_OTLP_ENDPOINT where appropriate.
	// e.g.
	// grpc: "http://localhost:4317"
	// http: "http://localhost:4318"
	shutdown, err := initHTTPTracerProvider(ctx, res, cfg)
	if err != nil {
		return nil, err
	}

	return shutdown, nil
}

func initHTTPTracerProvider(ctx context.Context, res *resource.Resource, cfg TracerConfig) (func(context.Context) error, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating otlptracehttp: %w", err)
	}

	sampler := trace.ParentBased(
		trace.TraceIDRatioBased(cfg.SampleRate),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithMaxQueueSize(2048),
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(5*time.Second),
		),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func NewDB(dsn string) (*sql.DB, error) {
	return otelsql.Open("postgres", dsn,
		otelsql.WithAttributes(semconv.DBSystemNamePostgreSQL),
	)
}
