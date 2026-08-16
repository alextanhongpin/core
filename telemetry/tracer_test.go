package telemetry

import "testing"

func TestTracer(t *testing.T) {
	NewTracer(t.Context(), TracerConfig{})
}
