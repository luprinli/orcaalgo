# /anti-pattern — Scan for Hard Prohibition Violations

Runs the automated anti-pattern scanner against all 10 hard prohibitions from the stack constitution.

## Steps

1. `python scripts/anti_pattern_scan.py` — Full scan (all 10 rules)
2. Report violations with file:line:spec reference
3. Suggest fixes for auto-fixable violations

## Severity Levels (NEW)

- CRITICAL: Must fix before merge (Rules 1, 2, 4, 6, 8)
- HIGH: Must fix before deploy (Rules 5, 7, 9, 10)
- MEDIUM: Advisory (Rule 3 legacy YAML in non-strategy paths)

## Exit Code

- 0: Zero violations (CRITICAL + HIGH)
- 1: Violations found (CRITICAL or HIGH)
- 2: Scan error

## SARIF Output

For CI dashboard integration: `python scripts/anti_pattern_scan.py --format sarif`
