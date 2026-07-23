---
name: emos-bias-correction
description: Apply EMOS / Platt scaling to correct systematic bias in a probabilistic forecast at inference time. Use when the user asks "fit EMOS", "apply Platt scaling", "calibrate my model output", "correct model bias", "my reliability diagram has a slope", "the model is over-confident", or has a calibration audit showing systematic deviation that a post-hoc transformation could fix. Also use when adding a new data source / cohort that may have its own bias profile relative to the existing fit.
---

# EMOS bias correction

Fit a post-hoc transformation that maps raw model probabilities to calibrated probabilities. The classical approach is two-parameter Platt scaling (a logistic regression on the model's logit output); the more general framework is EMOS (Ensemble Model Output Statistics), which generalizes to multi-parameter affine corrections on the predictive *distribution*.

For most retail quant use cases, Platt scaling on the output probability is the right starting point. Move to richer EMOS only when there's evidence that the spread of the predictive distribution itself is miscalibrated, not just its center.

## When to apply bias correction

Apply only when a calibration audit shows systematic, structured miscalibration:

- A consistent slope in the reliability diagram (model is over-confident or under-confident in a monotone way across all bins).
- An S-shape (over-confident at the extremes, under-confident in the middle).
- Different miscalibration per cohort, side, or category — fit one correction per cohort.

Do not apply when:

- The reliability diagram is noisy but unbiased (a transformation will overfit the noise).
- The model is fundamentally wrong (poor resolution). Calibration corrects the *probability scale*, not the underlying predictive power.
- You don't have a clean out-of-time validation slice to evaluate the fit on.

## Fitting Platt scaling

The two-parameter Platt fit is:

```
calibrated_p = sigmoid(A · logit(raw_p) + B)
```

where `A` and `B` are fit by minimizing log-loss (or Brier score) on a training set of `(raw_p, outcome)` pairs.

Implementation notes:

- Use a true out-of-time training/validation split. Fit `A, B` on the training slice, evaluate Brier improvement on the validation slice. If the validation Brier doesn't improve over the raw model, do not promote the fit.
- Clip `raw_p` to `[ε, 1−ε]` (typically `ε = 1e-6`) before taking the logit, to avoid infinities.
- Re-fit periodically. A monthly cadence is reasonable for stable strategies; weekly for fast-changing ones.

## Per-cohort fitting

Bias is often per-cohort (per market category, per geographic region, per data-source upstream). Fit `A, B` separately per cohort when:

- The cohort has at least 200 training observations.
- The validation Brier reduction is meaningfully larger with the cohort-specific fit than with the global fit.

Fall back to the global fit when a cohort has too few observations or the cohort-specific fit doesn't help. Maintain a `verdict` per cohort: `USE_COHORT_FIT`, `USE_GLOBAL_FIT`, or `USE_RAW` (the model is already calibrated).

## Beyond Platt: full EMOS

Platt scaling only adjusts the *mean* of the predictive distribution. Full EMOS (Gneiting et al. 2005) adjusts both mean and spread:

```
calibrated_mu    = a + b · model_mu
calibrated_sigma = sqrt(c + d · model_sigma²)
```

This matters when the model's *uncertainty* is itself miscalibrated — e.g., the model says "very confident" too often or its confidence intervals don't have the stated coverage. Use this when you have point forecasts plus a spread estimate (e.g., an ensemble of model runs), not just a single output probability.

For prediction-market trading, full EMOS is most relevant when blending multiple model sources where each source's spread is itself a function of model agreement.

## Workflow

1. Run a calibration audit (see `calibration-audit`) on the raw model output. Confirm there's structured miscalibration to correct.
2. Split data into training and out-of-time validation.
3. Fit Platt scaling per cohort with at least 200 observations.
4. Evaluate Brier on the validation slice — global raw, global Platt, cohort Platt.
5. Promote per-cohort: use the lowest-Brier option, with `USE_RAW` as a fallback when neither fit helps.
6. Audit again on a *third* holdout to verify the promotion didn't overfit.

## How to respond

When the user asks for an EMOS / Platt fit:

1. Confirm there's a calibration audit showing structured miscalibration. If not, run `calibration-audit` first.
2. Confirm there's enough data — at minimum 500 observations total, 200 per cohort for cohort-specific fits.
3. Run the fit (script reference below) on the training slice.
4. Report the verdicts per cohort (`USE_COHORT_FIT` / `USE_GLOBAL_FIT` / `USE_RAW`).
5. Show the Brier improvement on the validation slice. If under 5%, recommend leaving the production fit unchanged — the cost of redeploying isn't justified by noise-level improvement.
6. Write the fitted coefficients to a versioned file (with a date stamp) and keep the previous version for rollback.

## Reference script

A fitting helper is at `scripts/platt_fit.py`:

```
python scripts/platt_fit.py --train predictions_train.csv --val predictions_val.csv --by cohort
```

It computes per-cohort Platt fits, evaluates on the validation slice, writes a `coefficients.json` file with one block per cohort, and emits the verdict table.

## Don't

- Re-fit on the same slice you'll evaluate on. Overfit calibration looks great on training and is worse than nothing in production.
- Apply a single global fit to a multi-cohort model without checking per-cohort first. The global fit will mask per-cohort drift.
- Clobber the production coefficients without keeping a rollback copy. Calibration regressions are hard to detect quickly.
- Use Platt when the miscalibration is non-monotonic or genuinely nonlinear. Use isotonic regression instead.

## References

- Platt, J. (1999). *Probabilistic Outputs for Support Vector Machines and Comparisons to Regularized Likelihood Methods.*
- Gneiting, T., Raftery, A.E., Westveld, A.H., and Goldman, T. (2005). *Calibrated Probabilistic Forecasting Using Ensemble Model Output Statistics and Minimum CRPS Estimation.*
- Niculescu-Mizil & Caruana (2005). *Predicting Good Probabilities With Supervised Learning.* (Platt vs isotonic comparison.)
