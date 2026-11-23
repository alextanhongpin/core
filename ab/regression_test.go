package ab

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
)

func newEngineExp(t *testing.T, exp *Experiment) *ExperimentEngine {
	t.Helper()
	e := NewExperimentEngine(NewInMemoryExperimentStore())
	if err := e.CreateExperiment(exp); err != nil {
		t.Fatal(err)
	}
	return e
}

func twoArms() []Variant {
	return []Variant{{ID: "control", Weight: 0.5}, {ID: "treatment", Weight: 0.5}}
}

// 0.29*100 truncated to 28%; it now rounds to 29%.
func TestTargetPercentageRounds(t *testing.T) {
	e := newEngineExp(t, &Experiment{ID: "x", Status: StatusActive, Variants: twoArms(), TargetPercentage: 0.29})
	var included int
	const n = 100000
	for i := range n {
		a, err := e.AssignVariant(context.Background(), fmt.Sprint(i), "x", nil)
		if err != nil {
			t.Fatal(err)
		}
		if a != nil {
			included++
		}
	}
	if got := float64(included) / n; got < 0.285 || got > 0.295 {
		t.Fatalf("included %.4f, want about 0.29", got)
	}
}

func TestCreateExperimentValidates(t *testing.T) {
	for name, exp := range map[string]*Experiment{
		"no variants":    {TargetPercentage: 1},
		"zero target":    {Variants: twoArms()},
		"duplicate id":   {TargetPercentage: 1, Variants: []Variant{{ID: "a", Weight: .5}, {ID: "a", Weight: .5}}},
		"negative":       {TargetPercentage: 1, Variants: []Variant{{ID: "a", Weight: 1.5}, {ID: "b", Weight: -.5}}},
		"missing weight": {TargetPercentage: 1, Variants: []Variant{{ID: "a", Weight: .5}}},
	} {
		if err := NewExperimentEngine(nil).CreateExperiment(exp); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	e := newEngineExp(t, &Experiment{ID: "x", Variants: twoArms(), TargetPercentage: 1})
	if err := e.CreateExperiment(&Experiment{ID: "x", Variants: twoArms(), TargetPercentage: 1}); err == nil {
		t.Error("overwrote an existing experiment")
	}
}

// A zero-weight variant must never be selected, even for hash 0.
func TestSelectVariantSkipsZeroWeight(t *testing.T) {
	e := NewExperimentEngine(nil)
	vs := []Variant{{ID: "off", Weight: 0}, {ID: "on", Weight: 1}}
	for i := range 10000 {
		if got := e.selectVariant(fmt.Sprint(i), "x", vs); got != 1 {
			t.Fatalf("user %d got variant %d", i, got)
		}
	}
}

// Assignment keys were "user_exp" strings, so ("a_b", "c") and ("a", "b_c")
// shared one assignment.
func TestAssignmentKeyNoCollision(t *testing.T) {
	e := NewExperimentEngine(nil)
	for _, id := range []string{"c", "b_c"} {
		if err := e.CreateExperiment(&Experiment{ID: id, Status: StatusActive, Variants: twoArms(), TargetPercentage: 1}); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	a1, _ := e.AssignVariant(ctx, "a_b", "c", nil)
	a2, _ := e.AssignVariant(ctx, "a", "b_c", nil)
	if a2.UserID != "a" || a2.ExperimentID != "b_c" {
		t.Fatalf("got %+v for user a; first assignment %+v", a2, a1)
	}
}

// Conversions were counted per event, for any variant, so rates could
// exceed 1.
func TestRecordConversionCountsAssignedUsersOnce(t *testing.T) {
	e := newEngineExp(t, &Experiment{ID: "x", Status: StatusActive, Variants: twoArms(), TargetPercentage: 1})
	ctx := context.Background()

	if err := e.RecordConversion(ctx, &ConversionEvent{UserID: "u", ExperimentID: "x"}); err == nil {
		t.Fatal("converted an unassigned user")
	}

	a, _ := e.AssignVariant(ctx, "u", "x", nil)
	other := "control"
	if a.VariantID == other {
		other = "treatment"
	}
	if err := e.RecordConversion(ctx, &ConversionEvent{UserID: "u", ExperimentID: "x", VariantID: other}); err == nil {
		t.Fatal("accepted a conversion for the wrong variant")
	}
	for range 3 {
		if err := e.RecordConversion(ctx, &ConversionEvent{UserID: "u", ExperimentID: "x", VariantID: a.VariantID}); err != nil {
			t.Fatal(err)
		}
	}

	res, _ := e.GetExperimentResults("x")
	for _, v := range res.Variants {
		if v.ConversionRate > 1 {
			t.Fatalf("rate %v > 1", v.ConversionRate)
		}
		if v.VariantID == a.VariantID && v.Conversions != 1 {
			t.Fatalf("conversions = %d, want 1", v.Conversions)
		}
	}
}

// A zero ConfidenceLevel made every result with p < 1 significant.
func TestDefaultConfidenceLevel(t *testing.T) {
	e := newEngineExp(t, &Experiment{ID: "x", Status: StatusActive, Variants: twoArms(), TargetPercentage: 1})
	exp := e.experiments["x"]
	exp.Variants[0].Impressions, exp.Variants[0].Conversions = 1000, 100
	exp.Variants[1].Impressions, exp.Variants[1].Conversions = 1000, 110

	res, _ := e.GetExperimentResults("x")
	if v := res.Variants[1]; v.IsSignificant || v.PValue < 0.4 {
		t.Fatalf("p=%v significant=%v", v.PValue, v.IsSignificant)
	}
	// Wilson interval at 95% for 100/1000.
	if ci := res.Variants[0].ConfidenceInterval; math.Abs(ci.Lower-0.0829) > 1e-3 || math.Abs(ci.Upper-0.1203) > 1e-3 {
		t.Fatalf("ci = %+v", ci)
	}
}

func TestZScore(t *testing.T) {
	for c, want := range map[float64]float64{0.9: 1.6449, 0.95: 1.9600, 0.99: 2.5758, 0: 1.9600} {
		if got := zScore(c); math.Abs(got-want) > 1e-4 {
			t.Errorf("zScore(%v) = %v, want %v", c, got, want)
		}
	}
}

func TestExperimentEngineConcurrent(t *testing.T) {
	e := newEngineExp(t, &Experiment{ID: "x", Status: StatusActive, Variants: twoArms(), TargetPercentage: 1})
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 100 {
				u := fmt.Sprint(i, "-", j)
				if _, err := e.AssignVariant(ctx, u, "x", nil); err != nil {
					t.Error(err)
					return
				}
				_ = e.RecordConversion(ctx, &ConversionEvent{UserID: u, ExperimentID: "x"})
				_, _ = e.GetExperimentResults("x")
			}
		}()
	}
	wg.Wait()

	res, _ := e.GetExperimentResults("x")
	if n := res.Variants[0].Impressions + res.Variants[1].Impressions; n != 800 {
		t.Fatalf("impressions = %d, want 800", n)
	}
}

// sampleGamma returned Exp(1) for every shape <= 10, so Beta draws had mean
// 0.5 whatever the posterior.
func TestSampleBetaMean(t *testing.T) {
	b := NewBanditEngine()
	b.rng = rand.New(rand.NewSource(1))
	for _, tc := range []struct{ a, b float64 }{{2, 8}, {1, 1}, {0.5, 0.5}, {30, 10}} {
		const n = 20000
		sum := 0.0
		for range n {
			sum += b.sampleBeta(tc.a, tc.b)
		}
		want := tc.a / (tc.a + tc.b)
		if got := sum / n; math.Abs(got-want) > 0.01 {
			t.Errorf("Beta(%v, %v) mean = %.3f, want %.3f", tc.a, tc.b, got, want)
		}
	}
}

// Regret was measured against the best of the arms analyzed so far, so
// arms before the best one got zero or negative regret.
func TestBanditRegret(t *testing.T) {
	b := NewBanditEngine()
	exp := &BanditExperiment{ID: "x", Algorithm: UCB, Arms: []BanditArm{{ID: "low"}, {ID: "high"}}}
	if err := b.CreateBanditExperiment(exp); err != nil {
		t.Fatal(err)
	}
	exp.Arms[0].Pulls, exp.Arms[0].Rewards = 10, 1
	exp.Arms[1].Pulls, exp.Arms[1].Rewards = 10, 5

	a, err := b.GetBanditResults("x")
	if err != nil {
		t.Fatal(err)
	}
	if a.Arms[0].ArmID != "high" || a.Arms[0].Regret != 0 {
		t.Fatalf("best arm: %+v", a.Arms[0])
	}
	if got := a.Arms[1].Regret; math.Abs(got-4) > 1e-9 {
		t.Fatalf("low arm regret = %v, want 4", got)
	}
	if math.Abs(a.TotalRegret-4) > 1e-9 {
		t.Fatalf("total regret = %v, want 4", a.TotalRegret)
	}
}

func TestBanditValidation(t *testing.T) {
	for name, exp := range map[string]*BanditExperiment{
		"no arms":   {Algorithm: UCB},
		"dup arm":   {Algorithm: UCB, Arms: []BanditArm{{ID: "a"}, {ID: "a"}}},
		"epsilon":   {Algorithm: EpsilonGreedy, Epsilon: 2, Arms: []BanditArm{{ID: "a"}}},
		"algorithm": {Algorithm: "nope", Arms: []BanditArm{{ID: "a"}}},
	} {
		if err := NewBanditEngine().CreateBanditExperiment(exp); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestInMemoryBanditStorageListResults(t *testing.T) {
	s := NewInMemoryBanditStorage()
	for range 3 {
		_ = s.SaveResult(BanditResult{ExperimentID: "x"})
	}
	if rs, _ := s.ListResults("x", -1); len(rs) != 3 {
		t.Fatalf("limit -1: got %d", len(rs))
	}
	if rs, _ := s.ListResults("x", 2); len(rs) != 2 {
		t.Fatalf("limit 2: got %d", len(rs))
	}
}

func TestBanditEngineConcurrent(t *testing.T) {
	b := NewBanditEngine()
	exp := &BanditExperiment{ID: "x", Algorithm: ThompsonSampling, Arms: []BanditArm{{ID: "a"}, {ID: "b"}}}
	if err := b.CreateBanditExperiment(exp); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				arm, err := b.SelectArm("x", "u", nil)
				if err != nil {
					t.Error(err)
					return
				}
				_ = b.RecordReward("x", arm.ID, "u", 1, true, nil)
				_, _ = b.GetBanditResults("x")
				_ = b.Metrics()
				_ = b.Experiments()
			}
		}()
	}
	wg.Wait()
	if m := b.Metrics(); m.Pulls != 800 || m.Rewards != 800 {
		t.Fatalf("metrics = %+v", m)
	}
}
