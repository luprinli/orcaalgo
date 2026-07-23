# /deploy-gate — Full Pre-Deployment Verification

Runs the complete pre-deployment verification pipeline. Use before any production deployment. All gates must PASS.

## Prerequisites (GUARDRAIL)

0. **Environment safety check:** `python scripts/env_guard.py --check deploy_gate`
   - If blocked, STOP and report. Do NOT proceed.
1. **Change audit:** `python scripts/change_audit.py`
   - Verify no destructive changes are pending. If blocked, resolve before deploying.

## Steps

1. Delegates to `/preflight` (which includes its own env guard)
2. Delegates to `/calibrate` for calibration audit
3. GKR strategy hash verification: `orca validate configs/strategies/*.gkr.yaml --strict`
4. Kill-switch E2E test
5. Balance reconciliation
6. Teardown all services started during verification

## Exit Code

- 0: All gates PASS
- 1: Any gate FAILS
- 2: Environment guard blocked (live account detected)
