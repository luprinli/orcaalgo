---
name: calibration-audit
description: Audit how well-calibrated a probabilistic forecast is — does "70% confident" actually come true 70% of the time? Use when the user asks "is my model calibrated", "reliability diagram", "audit my predictions", "Brier score breakdown", "are my probabilities trustworthy", "should I trust this model's confidence", or supplies a history of (forecast_probability, outcome) pairs and wants to assess whether the model is over- or under-confident. Also use after a model refit to verify the new fit isn't worse than the old one.
---

# Calibration audit

Assess whether a probabilistic model's stated confidence matches its empirical accuracy. A model can be highly accurate (good Brier score) and still be miscalibrated (says 90% when the true rate is 70%), and miscalibration is what causes Kelly to over-bet.

## What "calibrated" means

A forecast is calibrated if, across all predictions where the model said `p = X`, the event actually occurs at rate `X`. Calibration is a property of the *probability distribution* the model outputs, not the model's accuracy. Two failure modes matter:

- **Over-confident** — model says 90% when reality is 70%. Causes systematic over-betting. The most common failure mode for retail quant models, because more training data plus more features pushes the model toward extreme probabilities the data doesn't actually support.
- **Under-confident** — model says 60% when reality is 80%. Leaves money on the table. Less dangerous than over-confidence but still costly.

Calibration is also direction-specific. A model may be well-calibrated on the YES side and badly calibrated on the NO side, or vice versa. Always audit both sides separately.

## The reliability diagram

The canonical visualization: bucket predictions by forecast probability (e.g., 10 bins of width 0.1), then for each bucket plot the bucket's mean forecast on the x-axis and the bucket's empirical hit rate on the y-axis. A perfectly calibrated model lies on the y = x diagonal.

Read the diagram by direction of deviation:

- Points below the diagonal in the high-probability bins — model is over-confident on bullish forecasts.
- Points above the diagonal in the low-probability bins — model is over-confident on bearish forecasts (saying "very unlikely" when it's actually unlikely-but-not-that-unlikely).
- Systematic S-shape — model is over-confident at the extremes and under-confident in the middle. This is what an uncalibrated raw classifier output usually looks like.

## Brier score and its decomposition

The Brier score is the mean squared error between forecast probability and the binary outcome. Lower is better. But the raw score conflates three different things, so always decompose it:

```
Brier = Reliability − Resolution + Uncertainty
```

- **Reliability** (lower is better) — squared distance from the calibration diagonal. Pure calibration error.
- **Resolution** (higher is better) — variance of the per-bucket hit rates around the base rate. Measures how much the model actually discriminates outcomes.
- **Uncertainty** (constant for a given dataset) — variance of the outcome itself, p̄·(1−p̄). Not a model property; just the difficulty of the underlying problem.

A model can have a great Brier score because the base rate is low (uncertainty is small), not because it's actually good. Resolution is the term that captures real predictive power.

## How to run an audit

When the user asks for a calibration audit:

1. Ask for the data: a list (or path to a CSV/SQLite) of `(forecast_p, outcome)` pairs. Outcomes must be 0/1. If they have YES vs NO sides, request both and audit separately.
2. Compute the reliability diagram with 10 bins by default (5 if n < 200, 20 if n > 5,000).
3. Compute Brier score and its three-way decomposition.
4. Identify which bins are most miscalibrated. Report the worst three.
5. Note the sample size *per bin*. Bins with fewer than 20 samples are unreliable — flag them, don't draw conclusions from them.
6. If the user has multiple model versions or strategies, audit each separately so they can compare.

## Decision rules

Once the audit is done, recommend an action:

- **All bins within ±5% of diagonal and reliability < 0.005** — model is well-calibrated. No action needed beyond monitoring.
- **Systematic S-shape, reliability between 0.005 and 0.02** — apply a calibration layer (Platt scaling or isotonic regression). See the `emos-bias-correction` skill.
- **Reliability > 0.02 or extreme over-confidence in any bin** — do not size positions based on this model's outputs. Either retrain with better regularization or apply a much smaller Kelly multiplier (⅛ or less) while you investigate.
- **Resolution near zero** — the model isn't discriminating outcomes. Calibration won't save it.

## When to re-audit

Calibration drifts. Re-audit on a rolling basis: every 30 days, every 100 new outcomes, or whenever upstream data sources change. Track the audit history so you can detect drift before it costs money.

## Reference script

A working audit is at `scripts/calibration_audit.py`:

```
python scripts/calibration_audit.py --csv predictions.csv --bins 10
```

It accepts a CSV with columns `forecast_p,outcome` (optional `side`, `cohort` for grouping) and prints the Brier decomposition + per-bin reliability table. Read it before implementing — the bin-edge handling matters at the boundaries.

## Don't

- Compare Brier scores across datasets. The uncertainty term changes; the scores aren't comparable.
- Trust calibration on the holdout set if your training data was the same period. Use a true out-of-time validation slice.
- Average forecast probabilities across cohorts before bucketing. That hides per-cohort miscalibration.

## References

- Brier, G.W. (1950). *Verification of Forecasts Expressed in Terms of Probability.*
- Gneiting & Raftery (2007). *Strictly Proper Scoring Rules, Prediction, and Estimation.*
- Murphy, A.H. (1973). *A New Vector Partition of the Probability Score.*
