package ab_test

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/alextanhongpin/core/ab"
)

func TestFixedTest(t *testing.T) {
	if _, err := ab.NewFixedTest(""); err == nil {
		t.Fatal("accepted empty ID")
	}
	e, _ := ab.NewFixedTest("<script>alert(1)</script>")
	if _, err := e.Expose(""); err == nil {
		t.Fatal("accepted empty user")
	}
	if err := e.Convert("unknown"); err == nil {
		t.Fatal("accepted unexposed conversion")
	}
	empty := e.Report()
	if empty.Ready || empty.PValue != 1 || empty.Variants[0].ConfidenceInterval.Upper != 1 {
		t.Fatal(empty)
	}
	counts := [2]int{}
	for i := 0; counts[0] < 1000 || counts[1] < 1000; i++ {
		user := fmt.Sprint(i)
		arm, err := e.Expose(user)
		if err != nil {
			t.Fatal(err)
		}
		j := 0
		if arm == "treatment" {
			j = 1
		}
		counts[j]++
		cutoff := 100
		if j == 1 {
			cutoff = 150
		}
		if counts[j] <= cutoff {
			if err := e.Convert(user); err != nil {
				t.Fatal(err)
			}
			e.Convert(user)
		}
		again, _ := e.Expose(user)
		if again != arm {
			t.Fatal("unstable assignment")
		}
	}
	r := e.Report()
	if !r.Ready || !r.Significant || r.Lift <= 0 || r.PValue <= 0 || r.PValue >= .05 {
		t.Fatal(r)
	}
	for _, v := range r.Variants {
		if v.Conversions > v.Impressions || v.ConfidenceInterval.Lower >= v.ConversionRate || v.ConfidenceInterval.Upper <= v.ConversionRate {
			t.Fatal(v)
		}
	}
	var out bytes.Buffer
	if err := r.WriteHTML(&out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>") || !strings.Contains(out.String(), "&lt;script&gt;") {
		t.Fatal("unescaped ID")
	}
	r.Variants[0].Impressions = 0
	if e.Report().Variants[0].Impressions == 0 {
		t.Fatal("shared snapshot")
	}
}

func TestFixedConcurrent(t *testing.T) {
	e, _ := ab.NewFixedTest("concurrent")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); e.Expose("same"); e.Convert("same"); e.Report() }()
	}
	wg.Wait()
	r := e.Report()
	if r.Variants[0].Impressions+r.Variants[1].Impressions != 1 || r.Variants[0].Conversions+r.Variants[1].Conversions != 1 {
		t.Fatal(r)
	}
}

func TestFixedKnownPValue(t *testing.T) {
	e, _ := ab.NewFixedTest("known")
	counts := [2]int{}
	for i := 0; counts[0] < 1000 || counts[1] < 1000; i++ {
		user := fmt.Sprint(i)
		// Find enough users in each arm without adding extra exposures.
		arm := ab.Hash(fmt.Sprintf("%d:%s%s", len("known"), "known", user), 2)
		if counts[arm] >= 1000 {
			continue
		}
		e.Expose(user)
		counts[arm]++
		target := 100
		if arm == 1 {
			target = 150
		}
		if counts[arm] <= target {
			e.Convert(user)
		}
	}
	r := e.Report()
	if math.Abs(r.PValue-0.0007232327164301936) > 1e-10 {
		t.Fatal(r.PValue)
	}
}

func TestReportInterpretation(t *testing.T) {
	cases := []struct {
		name   string
		report ab.FixedReport
		want   string
	}{
		{"insufficient", ab.FixedReport{}, "Too little data to interpret significance"},
		{"inconclusive", ab.FixedReport{Ready: true}, "Inconclusive: no statistically detectable difference"},
		{"higher", ab.FixedReport{Ready: true, Significant: true, Lift: .02}, "Treatment has a statistically higher conversion rate"},
		{"lower", ab.FixedReport{Ready: true, Significant: true, Lift: -.02}, "Treatment has a statistically lower conversion rate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tc.report.WriteHTML(&out); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.want) || !strings.Contains(out.String(), "Best practices") {
				t.Fatal("missing interpretation or guide")
			}
			if tc.name == "higher" && !strings.Contains(html.UnescapeString(out.String()), "+2.00 percentage points") {
				t.Fatal("incorrect lift unit")
			}
		})
	}
}
