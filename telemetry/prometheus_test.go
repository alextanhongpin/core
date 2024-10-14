package telemetry_test

import (
	"context"
	"testing"

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
	metric := telemetry.NewPrometheusHandler(prometheus.NewRegistry())
	ctx = event.WithExporter(ctx, event.NewExporter(&telemetry.MultiHandler{
		Metric: metric,
	}, eventtest.ExporterOptions()))
	c := event.NewCounter("hits", &event.MetricOptions{Description: "Earth meteorite hits"})
	c.Record(ctx, 123, event.String("version", "stable"))
	c.Record(ctx, 456, event.String("version", "canary"))

	collector := metric.Collector("hits")

	is := assert.New(t)
	is.Equal(2, testutil.CollectAndCount(collector, "hits"))
	b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "hits")
	is.Nil(err)
	want := `# HELP hits Earth meteorite hits
# TYPE hits counter
hits{version="canary"} 456
hits{version="stable"} 123
`
	is.Equal(want, string(b))
}
