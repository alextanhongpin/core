package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
)

var ctx = context.Background()

func TestPrometheus(t *testing.T) {
	t.Run("counter", func(t *testing.T) {
		metric := telemetry.NewPrometheusHandler(prometheus.NewRegistry())
		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		c := event.NewCounter("hits", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "Earth meteorite hits"},
		)
		c.Record(ctx, 123, event.String("version", "stable"))
		c.Record(ctx, 456, event.String("version", "canary"))

		collector, ok := metric.Collector("hits")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_hits"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_hits")
		is.NoError(err)
		want := `# HELP my_ns_hits Earth meteorite hits
# TYPE my_ns_hits counter
my_ns_hits{version="canary"} 456
my_ns_hits{version="stable"} 123
`
		is.Equal(want, string(b))
	})

	t.Run("gauge", func(t *testing.T) {
		metric := telemetry.NewPrometheusHandler(prometheus.NewRegistry())
		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		g := event.NewFloatGauge("cpu", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "cpu usage"},
		)
		g.Record(ctx, 123, event.String("version", "canary"))
		g.Record(ctx, 456, event.String("version", "canary"))
		g.Record(ctx, 456, event.String("version", "stable"))
		g.Record(ctx, 123, event.String("version", "stable"))

		collector, ok := metric.Collector("cpu")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_cpu"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_cpu")
		is.NoError(err)
		want := `# HELP my_ns_cpu cpu usage
# TYPE my_ns_cpu gauge
my_ns_cpu{version="canary"} 456
my_ns_cpu{version="stable"} 123
`
		is.Equal(want, string(b))
	})

	t.Run("histogram", func(t *testing.T) {
		metric := telemetry.NewPrometheusHandler(prometheus.NewRegistry())
		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		h := event.NewDuration("request_duration", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "request per seconds",
			//Unit:        event.UnitMilliseconds,
		})
		h.Record(ctx, time.Second, event.String("version", "stable"))
		h.Record(ctx, time.Minute, event.String("version", "canary"))

		collector, ok := metric.Collector("request_duration")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_request_duration_seconds"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_request_duration_seconds")
		is.NoError(err)
		want := `# HELP my_ns_request_duration_seconds request per seconds
# TYPE my_ns_request_duration_seconds histogram
my_ns_request_duration_seconds_bucket{version="canary",le="0.005"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.01"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.025"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.05"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.1"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.25"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="1"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="2.5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="10"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="+Inf"} 1
my_ns_request_duration_seconds_sum{version="canary"} 60
my_ns_request_duration_seconds_count{version="canary"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="0.005"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.01"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.025"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.05"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.1"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.25"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.5"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="1"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="2.5"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="5"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="10"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="+Inf"} 1
my_ns_request_duration_seconds_sum{version="stable"} 1
my_ns_request_duration_seconds_count{version="stable"} 1
`
		is.Equal(want, string(b))
	})
}
