package ab

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"sync"
)

// FixedTest measures one binary conversion per exposed user in a fixed 50/50
// control/treatment experiment. It is safe for concurrent use. State is in memory;
// retain the instance for the experiment's lifetime. Do not copy it.
type FixedTest struct {
	mu     sync.Mutex
	id     string
	users  map[string]bool
	counts [2]VariantResults
}

// NewFixedTest creates a fixed 50/50 experiment with stable user assignments.
func NewFixedTest(id string) (*FixedTest, error) {
	if id == "" {
		return nil, fmt.Errorf("ab: experiment ID is required")
	}
	return &FixedTest{id: id, users: make(map[string]bool), counts: [2]VariantResults{{VariantID: "control"}, {VariantID: "treatment"}}}, nil
}

func (t *FixedTest) arm(user string) int {
	if Hash(fmt.Sprintf("%d:%s%s", len(t.id), t.id, user), 2) == 0 {
		return 0
	}
	return 1
}

// Expose assigns a user and counts their first exposure. Call when the user
// actually sees the variant. Repeated exposures do not increase the sample size.
func (t *FixedTest) Expose(user string) (string, error) {
	if user == "" {
		return "", fmt.Errorf("ab: user ID is required")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	arm := t.arm(user)
	if _, ok := t.users[user]; !ok {
		t.users[user] = false
		t.counts[arm].Impressions++
	}
	return t.counts[arm].VariantID, nil
}

// Convert records the exposed user's first conversion; retries are idempotent.
func (t *FixedTest) Convert(user string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	converted, ok := t.users[user]
	if !ok {
		return fmt.Errorf("ab: user has not been exposed")
	}
	if !converted {
		t.users[user] = true
		t.counts[t.arm(user)].Conversions++
	}
	return nil
}

// FixedReport is a snapshot at 95% confidence. Lift is treatment minus control;
// RelativeLift is absent when the control rate is zero. Ready requires at least
// ten conversions and ten non-conversions per arm for the normal approximation.
// Significance is a two-sided pooled proportion z-test for a preplanned endpoint,
// not a sequential stopping rule. Confidence intervals use Wilson's method.
type FixedReport struct {
	ExperimentID string            `json:"experiment_id"`
	Variants     [2]VariantResults `json:"variants"`
	Lift         float64           `json:"lift"`
	RelativeLift *float64          `json:"relative_lift,omitempty"`
	PValue       float64           `json:"p_value"`
	Ready        bool              `json:"ready"`
	Significant  bool              `json:"significant"`
}

// Report returns an independent snapshot without exposing mutable internal state.
func (t *FixedTest) Report() FixedReport {
	t.mu.Lock()
	defer t.mu.Unlock()
	r := FixedReport{ExperimentID: t.id, Variants: t.counts, PValue: 1, Ready: true}
	for i := range r.Variants {
		v := &r.Variants[i]
		n := float64(v.Impressions)
		if n == 0 {
			v.ConfidenceInterval = ConfidenceInterval{Upper: 1}
		} else {
			p := float64(v.Conversions) / n
			v.ConversionRate = p
			z := 1.959963984540054
			z2 := z * z
			d := 1 + z2/n
			center := (p + z2/(2*n)) / d
			margin := z * math.Sqrt(p*(1-p)/n+z2/(4*n*n)) / d
			v.ConfidenceInterval = ConfidenceInterval{Lower: math.Max(0, center-margin), Upper: math.Min(1, center+margin)}
		}
		r.Ready = r.Ready && v.Conversions >= 10 && v.Impressions-v.Conversions >= 10
	}
	a, b := r.Variants[0], r.Variants[1]
	r.Lift = b.ConversionRate - a.ConversionRate
	if a.ConversionRate > 0 {
		lift := r.Lift / a.ConversionRate
		r.RelativeLift = &lift
	}
	if r.Ready {
		p := float64(a.Conversions+b.Conversions) / float64(a.Impressions+b.Impressions)
		se := math.Sqrt(p * (1 - p) * (1/float64(a.Impressions) + 1/float64(b.Impressions)))
		r.PValue = math.Erfc(math.Abs(r.Lift/se) / math.Sqrt2)
	}
	r.Significant = r.Ready && r.PValue < 0.05
	r.Variants[1].PValue = r.PValue
	r.Variants[1].IsSignificant = r.Significant
	return r
}

// WriteHTML writes a self-contained, escaped report with conversion-rate bars
// and confidence intervals. No network resources or JavaScript are required.
func (r FixedReport) WriteHTML(w io.Writer) error { return fixedHTML.Execute(w, r) }

var fixedHTML = template.Must(template.New("report").Funcs(template.FuncMap{
	"pct":   func(v float64) string { return fmt.Sprintf("%.2f%%", v*100) },
	"pp":    func(v float64) string { return fmt.Sprintf("%+.2f", v*100) },
	"width": func(v float64) float64 { return v * 600 },
}).Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>A/B report</title>
<style>body{font:16px system-ui;max-width:850px;margin:40px auto;padding:20px;color:#172238}svg{width:100%;height:60px}table{width:100%;border-collapse:collapse}td,th{text-align:left;padding:12px;border-bottom:1px solid #ddd}</style>
<h1>A/B report: {{.ExperimentID}}</h1><p>Fixed 50/50 assignment · Unique exposed users · 95% Wilson confidence intervals</p>
{{range .Variants}}<h2>{{.VariantID}}: {{pct .ConversionRate}}</h2><svg viewBox="0 0 600 60" role="img" aria-label="{{.VariantID}} conversion rate {{pct .ConversionRate}}, confidence interval {{pct .ConfidenceInterval.Lower}} to {{pct .ConfidenceInterval.Upper}}"><rect width="600" height="25" y="10" fill="#eef1f6"/><rect width="{{width .ConversionRate}}" height="25" y="10" fill="#4263eb"/><path d="M {{width .ConfidenceInterval.Lower}} 40 H {{width .ConfidenceInterval.Upper}}" stroke="#172238" stroke-width="4"/></svg>{{end}}
<table><caption>Conversion measurements</caption><tr><th>Variant</th><th>Exposures</th><th>Conversions</th><th>Rate</th><th>95% interval</th></tr>{{range .Variants}}<tr><td>{{.VariantID}}</td><td>{{.Impressions}}</td><td>{{.Conversions}}</td><td>{{pct .ConversionRate}}</td><td>{{pct .ConfidenceInterval.Lower}}–{{pct .ConfidenceInterval.Upper}}</td></tr>{{end}}</table>
<p>Absolute lift: {{pp .Lift}} percentage points. Relative lift: {{if .RelativeLift}}{{pct .RelativeLift}}{{else}}unavailable{{end}}.</p>
{{if .Ready}}<p>Two-sided p-value: {{printf "%.6g" .PValue}}. Significant at 5%: {{.Significant}}.</p>{{else}}<p>Insufficient data for the normal approximation: each arm needs at least 10 conversions and 10 non-conversions.</p>{{end}}
<section aria-labelledby="interpretation"><h2 id="interpretation">What does this result mean?</h2>
{{if not .Ready}}<p><strong>Too little data to interpret significance.</strong> Rates are descriptive. Complete planned collection; the readiness threshold is not a sample-size or power plan.</p>
{{else if not .Significant}}<p><strong>Inconclusive: no statistically detectable difference.</strong> This does not prove equal performance. Do not choose a winner from observed lift alone.</p>
{{else if gt .Lift 0.0}}<p><strong>Treatment has a statistically higher conversion rate.</strong> If conversions are desirable, this supports treatment at the planned endpoint. Check practical value and guardrails before shipping.</p>
{{else}}<p><strong>Treatment has a statistically lower conversion rate.</strong> If conversions are desirable, this favors control at the planned endpoint. If the event is undesirable, such as an error, lower may be beneficial.</p>{{end}}
<p>This report cannot verify your planned endpoint, sample-size target, telemetry quality, allocation balance or guardrails. Validate them before deciding.</p></section>
<section aria-labelledby="guide"><h2 id="guide">How to read this report</h2><dl>
<dt>Exposures and conversions</dt><dd>Unique exposed users and those who completed the goal at least once. Repeat visits and purchases count once.</dd>
<dt>Rate and lift</dt><dd>100 conversions among 1,000 users = 10%. Moving from 10% to 12% is +2 percentage points absolute lift, or +20% relative lift: two extra conversions per 100 users in the observed sample. Relative lift is unavailable for a zero control rate.</dd>
<dt>Confidence intervals</dt><dd>Dark lines show uncertainty in each arm's rate. Narrower intervals mean greater precision. Across repeated experiments, about 95% of intervals constructed this way cover the true rate. These are not intervals for lift; their overlap is not the significance test.</dd>
<dt>P-value</dt><dd>Assuming equal underlying rates and the test assumptions, the probability of a difference at least as extreme as observed in either direction. Not the probability treatment is better or that the result is due to chance. Below 0.05 meets this report's significance threshold.</dd>
<dt>Ready</dt><dd>Only checks ten conversions and ten non-conversions per arm for the normal approximation. It does not mean adequate power or a finished test. When false, JSON p=1 is a placeholder.</dd></dl></section>
<section aria-labelledby="practice"><h2 id="practice">Best practices</h2><ol>
<li>Choose one primary binary goal, its desired direction and the minimum improvement worth shipping. Plan sample size from the baseline rate, effect size, significance level and desired power.</li>
<li>Set the endpoint and conversion window before launch. Cover relevant business cycles and let delayed outcomes mature equally. This API does not enforce time windows.</li>
<li>Use stable IDs, one arm per user, concurrent variants and symmetric exposure logging. Investigate unexpected allocation imbalance with a sample-ratio-mismatch test; small deviations from 50/50 are normal.</li>
<li>Monitor tracking and safety, but do not stop for success when p first drops below 0.05. Early success decisions require sequential testing methods.</li>
<li>Check guardrails such as errors, latency and refunds separately. Significance alone does not establish business value.</li>
<li>Account for multiple metrics, segments and comparisons. Treat unplanned subgroup findings as exploratory and confirm them in a new test.</li>
<li>Persist measurements and experiment metadata outside this in-memory API; restarting loses state. Record the decision and monitor outcomes after rollout.</li>
</ol><p>Further reading: <a href="https://www.microsoft.com/en-us/research/articles/patterns-of-trustworthy-experimentation-during-experiment-stage/">Microsoft experimentation guidance</a> and <a href="https://www.itl.nist.gov/div898/handbook/prc/section2/prc241.htm">NIST confidence intervals</a>.</p></section></html>`))
