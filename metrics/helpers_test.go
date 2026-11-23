package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func histogram(t *testing.T, o prometheus.Observer) *dto.Histogram {
	t.Helper()

	var m dto.Metric
	if err := o.(prometheus.Metric).Write(&m); err != nil {
		t.Fatal(err)
	}
	return m.GetHistogram()
}

func histogramCount(t *testing.T, o prometheus.Observer) uint64 {
	t.Helper()
	return histogram(t, o).GetSampleCount()
}

func histogramSum(t *testing.T, o prometheus.Observer) float64 {
	t.Helper()
	return histogram(t, o).GetSampleSum()
}
