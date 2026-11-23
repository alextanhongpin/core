# ab

A/B testing primitives: stable hashing and rollouts, a fixed 50/50 conversion
test with an HTML report, a weighted multi-variant experiment engine, and
multi-armed bandits. All state is in memory and every type is safe for
concurrent use.

```go
import "github.com/alextanhongpin/core/ab"
```

## Hashing and rollouts

```go
bucket := ab.Hash("user-42", 10)           // stable value in [0, 10)
enabled := ab.Rollout("new-checkout:user-42", 20) // first 20 of 100 buckets
```

Prefix the key with the feature or experiment name so that different rollouts
pick independent users.

## Fixed 50/50 conversion test

`NewFixedTest` is the recommended way to run a two-arm conversion experiment.
Call `Expose` when the variant is actually seen and `Convert` for the binary
goal (purchase, signup, click). Both deduplicate by user; conversions before
exposure are rejected.

```go
experiment, err := ab.NewFixedTest("checkout-v2")
if err != nil { return err }
variant, err := experiment.Expose(userID) // "control" or "treatment"
if err != nil { return err }
// On the goal event:
if err := experiment.Convert(userID); err != nil { return err }

report := experiment.Report() // JSON-serializable snapshot
return report.WriteHTML(w)
```

Reports include unique exposures, conversions, rates, 95% Wilson intervals,
absolute and relative lift, and a two-sided pooled two-proportion z-test.
Significance is withheld until each arm has at least ten conversions and ten
non-conversions. Fix the endpoint before collecting data: stopping when the
p-value first drops below 0.05 invalidates the fixed-horizon test.

```sh
go run ./example/fixed > report.html # synthetic demo report
```

See [Interpreting results](INTERPRETING_RESULTS.md) for field definitions and a
decision table. Formulas follow NIST for
[Wilson intervals](https://www.itl.nist.gov/div898/handbook/prc/section2/prc241.htm)
and [two-proportion tests](https://www.itl.nist.gov/div898/software/dataplot/refman2/auxillar/diffprop.htm).

## Experiment engine

`ExperimentEngine` supports more than two weighted variants, a target
percentage and attribute filters.

```go
engine := ab.NewExperimentEngine(ab.NewInMemoryExperimentStore()) // store may be nil
err := engine.CreateExperiment(&ab.Experiment{
	ID:               "button_color",
	Status:           ab.StatusActive,
	Variants:         []ab.Variant{{ID: "blue", Weight: 0.5}, {ID: "red", Weight: 0.5}},
	TargetPercentage: 1.0,                           // (0, 1], rounded to whole percent
	Filters:          map[string]string{"country": "US"},
	ConfidenceLevel:  0.95,                          // default 0.95
})

a, err := engine.AssignVariant(ctx, userID, "button_color", attrs) // nil, nil if not targeted
err = engine.RecordConversion(ctx, &ab.ConversionEvent{UserID: userID, ExperimentID: "button_color"})
results, err := engine.GetExperimentResults("button_color")
```

- Variant IDs must be unique and weights non-negative, summing to 1.
- An impression is a unique assigned user; a conversion is counted once per
  assigned user and must match the user's variant. Every conversion event is
  still passed to the store.
- Results give Wilson intervals and a pooled z-test of each variant against
  the first one, without correction for multiple comparisons.
- The engine owns the `*Experiment` passed to `CreateExperiment`; change its
  status with `SetStatus`.

## Bandits

`BanditEngine` adaptively allocates traffic with epsilon-greedy, UCB1 or
Thompson sampling (`BayesianBandit` is an alias for Thompson sampling).

```go
engine := ab.NewBanditEngine()
err := engine.CreateBanditExperiment(&ab.BanditExperiment{
	ID:        "banner",
	Algorithm: ab.ThompsonSampling,
	Arms:      []ab.BanditArm{{ID: "a"}, {ID: "b"}, {ID: "c"}},
})

arm, err := engine.SelectArm("banner", userID, nil)
err = engine.RecordReward("banner", arm.ID, userID, 1, true, nil)

analysis, err := engine.GetBanditResults("banner")          // arms sorted by rate
rec, err := engine.GetExperimentRecommendation("banner")    // "continue" or "stop"
```

Call `RecordReward` once per `SelectArm`. Rates are rewards / pulls, regret is
empirical (best observed rate minus the arm's rate, times its pulls), and arm
intervals are a normal approximation of the Beta posterior. Bandit allocation
is not a substitute for a fixed-horizon significance test.
