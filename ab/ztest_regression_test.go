package ab

import (
	"math"
	"testing"
)

func TestZTestNormalTail(t *testing.T) {
	e := NewExperimentEngine(nil)
	p, err := e.calculateZTest(VariantResults{Impressions: 1000, Conversions: 100, ConversionRate: .1}, VariantResults{Impressions: 1000, Conversions: 150, ConversionRate: .15})
	if err != nil || math.Abs(p-0.0007232327164301936) > 1e-10 {
		t.Fatalf("p=%g err=%v", p, err)
	}
}
