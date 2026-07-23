# /ci — Run Full CI Pipeline Locally

Executes all quality gates in the same order as the GitHub Actions CI pipeline. Reads thresholds from `config/quality-gates.yaml`.

## Prerequisites (GUARDRAIL)

0. **Change audit:** `python scripts/change_audit.py`
   - Verify no unreviewed destructive changes. If blocked, resolve before running full CI.

## Steps (12 gates)

1. Go: `golangci-lint run ./...`
2. Go: `go vet ./...`
3. Go: `go test -race -count=1 -coverprofile=coverage.out ./internal/...`
4. Python: `ruff check orca/ tests/`
5. Python: `mypy orca/`
6. Python: `pytest tests/ -v --cov=orca --cov-fail-under=80` (threshold from quality-gates.yaml)
7. Odin: Build compilation check
8. Web: `cd web && npx tsc --noEmit && npx vite build`
9. GKR: `orca validate configs/strategies/*.gkr.yaml`
10. Anti-pattern: `python scripts/anti_pattern_scan.py` — zero violations
11. Guardian: `pytest tests/guardian/ -v`
12. Report summary with pass/fail per gate

## Exit Code

- 0: All 12 gates PASS
- 1: One or more gates FAIL
