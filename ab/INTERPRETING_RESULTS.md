# Interpreting A/B results

Read results in this order: data quality, uncertainty, then business value.

## What the numbers mean

| Measurement | Interpretation |
| --- | --- |
| Exposures | Unique users who saw the variant. Repeat visits count once. |
| Conversions | Exposed users who completed the binary goal at least once. Repeat purchases count once; this does not measure revenue. |
| Conversion rate | Conversions / exposures. 100 conversions among 1,000 users = 10%. |
| Absolute lift | Treatment rate minus control rate. Moving from 10% to 12% is +2 percentage points: two extra conversions per 100 users in the observed sample. |
| Relative lift | Absolute lift / control rate. Moving from 10% to 12% is +20%. Undefined when the control rate is zero. |
| 95% confidence interval | Uncertainty in each arm's rate. Across repeated experiments, approximately 95% of intervals constructed this way cover the true rate. |
| P-value | Assuming equal underlying rates and the test assumptions, probability of observing a difference at least as extreme in either direction. Not the probability treatment is better or that the result is due to chance. |
| Ready | At least ten conversions and ten non-conversions per arm. Only an approximation check, not adequate power or experiment completion. When false, p=1 is a placeholder. |
| Significant | Ready and two-sided p < 0.05. Can favor either arm; does not measure business value. |

The chart bars show rates on a shared 0–100% scale. Dark lines show per-arm Wilson
confidence intervals; narrower intervals mean greater precision. These are not
intervals for lift. Their overlap is not the significance test.

## Decide at the planned endpoint

| Result | Meaning | Action |
| --- | --- | --- |
| Not ready | Too little data for this approximation | Complete planned collection; rare outcomes may require an exact test outside this API. |
| Ready, not significant | Inconclusive: no statistically detectable difference | Do not declare equal performance or choose a winner from lift alone. |
| Significant, positive lift | Treatment has a statistically higher rate | For a desirable goal, consider treatment after checking practical value and guardrails. |
| Significant, negative lift | Treatment has a statistically lower rate | For a desirable goal, this favors control. For undesirable events such as errors, lower may be beneficial. |

Example: control has 100/1,000 conversions (10%) and treatment has 150/1,000
(15%). Absolute lift is +5 percentage points, relative lift is +50%, and p is
approximately 0.000723. At a preplanned endpoint, this supports a difference under
the test assumptions. It does not mean a 99.93% probability treatment is better,
or guarantee the improvement will repeat.

A p-value of 0.20 does not prove no effect: the sample may be too small. Establishing
equivalence or proving a minimum practical improvement requires another analysis.
This report provides neither a lift interval nor an equivalence test.

## Best practices

Before launch:

- Choose one primary binary goal, its desired direction, eligibility and a
  consistent conversion window. Define the smallest improvement worth shipping.
- Plan sample size using baseline rate, minimum detectable effect, significance
  level and desired power (often 80% or 90%). Ten events per arm is not a power
  calculation; this package does not calculate sample size.
- Set the endpoint in advance, cover relevant business cycles and let delayed
  outcomes mature equally in both arms. Do not extend a test just because p is
  close to 0.05.
- Define guardrails (errors, latency, refunds) and safety stop rules. Consider an
  A/A test to validate assignment and telemetry.

During the run:

- Use stable user IDs, one arm per user and concurrent control/treatment traffic.
  Log exposure at the same point in both experiences. Avoid eligibility based on
  treatment-affected behavior. If users influence each other, individual
  randomization and this independent-user test may be inappropriate.
- Investigate missing events and unexpected allocation imbalance. A 50/50 split
  need not have exactly equal counts; use a sample-ratio-mismatch test. The report
  does not run this data-quality check.
- Monitor tracking and safety, but do not stop for success when p first drops
  below 0.05. Early success decisions require sequential testing methods; this
  API uses a fixed endpoint.
- Keep variants, metrics and eligibility stable. Account for multiple metrics,
  segments and comparisons. Treat unplanned subgroup findings as exploratory.

After the endpoint:

- Validate tracking and guardrails, then compare effect size with business value
  and implementation cost. Statistical significance alone is not a ship decision.
- Archive counts, settings, dates and the decision. Confirm important exploratory
  findings in a new experiment and monitor outcomes after rollout.
- Persist measurements outside this API. `FixedTest` loses state on restart and
  does not enforce conversion windows, verify the endpoint, check allocation
  balance or adjust multiple comparisons.

References: [NIST confidence intervals](https://www.itl.nist.gov/div898/handbook/prc/section2/prc241.htm),
[NIST proportion tests](https://www.itl.nist.gov/div898/software/dataplot/refman2/auxillar/diffprop.htm),
[Microsoft experiment monitoring](https://www.microsoft.com/en-us/research/articles/patterns-of-trustworthy-experimentation-during-experiment-stage/),
and [sample ratio mismatch](https://www.microsoft.com/en-us/research/articles/diagnosing-sample-ratio-mismatch-in-a-b-testing/).
