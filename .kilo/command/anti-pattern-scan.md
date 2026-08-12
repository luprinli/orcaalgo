# /anti-pattern — Scan for Hard Prohibition Violations

Runs the automated anti-pattern scanner against all 18 hard prohibitions from the stack constitution.

## Steps

1. `python scripts/anti_pattern_scan.py` — Full scan (all 18 rules including Rule 11 HP #17 CI check)
2. Report violations with file:line:spec reference
3. Suggest fixes for auto-fixable violations

## Severity Levels

- CRITICAL: Must fix before merge (Rules 1, 2, 4, 6, 8, 11)
- HIGH: Must fix before deploy (Rules 5, 7, 9, 10)
- MEDIUM: Advisory (Rule 3 legacy YAML in non-strategy paths)

## Rule 11 — RiskPipeline Bypass Detection (HP #17)

Rule 11 is a CI-enforced check that scans Go files for `NewEngine()` calls followed by `Run()` without an intervening `WirePipeline()`. This ensures every backtest engine routes signals through the canonical `RiskPipeline.ProcessSignal` path. Violations are CRITICAL.

## Exit Code

- 0: Zero violations (CRITICAL + HIGH)
- 1: Violations found (CRITICAL or HIGH)
- 2: Scan error

## Output Formats

```bash
# Text output (default)
python scripts/anti_pattern_scan.py

# Text output only changed files
python scripts/anti_pattern_scan.py --changed-only

# Filter by minimum severity
python scripts/anti_pattern_scan.py --min-severity CRITICAL

# SARIF output for CI dashboards
python scripts/anti_pattern_scan.py --format sarif
```