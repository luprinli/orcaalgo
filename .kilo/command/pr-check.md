# /pr-check — Pre-Pull Request Quality Gates

Lightweight check for PR readiness. Runs only on changed files via test-related.

## Prerequisites (GUARDRAIL)

0. **Change audit:** `python scripts/change_audit.py`
   - If blocked (file count delta or line deletion % exceeds threshold), resolve before opening PR.
   - Critical file modifications require explicit review annotation in PR description.

## Steps

1. Detect changed files vs main (`git diff origin/main...HEAD --name-only`)
2. Run linters only on changed languages
3. Run `python scripts/test_related.py` — exit code 2 (not 0) if zero tests map to changed files
4. Run `python scripts/anti_pattern_scan.py` — zero violations required
5. If changed files include `internal/strategy/`, `internal/risk/`, or `orca/preflight/`, run full `/ci` pipeline
6. Report pass/fail with explicit violation locations and file paths

## Exit Code

- 0: All checks PASS, PR ready
- 1: Violations found — fix before PR
- 2: No tests found for changed files — add test coverage before PR
