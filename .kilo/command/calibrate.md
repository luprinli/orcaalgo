# /calibrate — Run Calibration Audit

Quarterly calibration audit on probability-emitting models.

## Prerequisites (GUARDRAIL)

0. **Environment safety check:** `python scripts/env_guard.py --check orca_calibrate`
   - Verify `PAPER_TRADING=true` if connecting to any broker for candle data.
1. **Data source verification:** Calibration must use real market data unless `--allow-synthetic` is explicitly passed.
   - Running against synthetic data without `--allow-synthetic` produces WARN result, not PASS.

## Steps

1. `orca calibrate --latest-only` — Run calibration audit
2. Check Brier score decomposition
3. Generate reliability diagram
4. Flag miscalibrated segments
5. Output report to `reports/calibration/`

## Exit Code

- 0: All models within tolerance
- 1: Requires recalibration
- 2: Environment guard blocked
