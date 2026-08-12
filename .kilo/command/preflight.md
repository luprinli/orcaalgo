# /preflight — Run Pre-Deployment Checklist

Runs the full 24-point pre-flight checklist from `docs/unified-best-practice-specifications.md` §9.3.

## Prerequisites (GUARDRAIL)

0. **Environment safety check:** `python scripts/env_guard.py --check preflight_strict`
   - If blocked (PAPER_TRADING != "true"), STOP and report. Do NOT proceed.
   - Only proceed if the check returns SAFE.

## Steps

1. `orca preflight --strict` — Run all 24 pre-flight checks
2. `orca validate configs/strategies/*.gkr.yaml --strict` — Verify GKR strategy hashes
3. `orca validate-data-integrity` — Cross-pipeline data integrity (VIX/vol, regime transitions, BPD, table alignment)
4. Kill-switch E2E: start server, trigger kill-switch, verify positions closed
5. Config integrity: verify `configs/config.prod.yaml` hash matches VCS tag
6. Balance reconciliation: verify ledger matches exchange balance
7. Report all items with PASS/FAIL/WARN status
8. Teardown: stop any servers started in step 4

## Exit Code

- 0: All gates PASS
- 1: Any CRITICAL gate FAILS
- 2: Environment guard blocked execution (live account detected)
