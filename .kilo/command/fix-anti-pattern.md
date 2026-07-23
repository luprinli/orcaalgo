# /fix-anti-pattern — Auto-Fix Common Anti-Pattern Violations

Attempts automatic fixes for mechanically fixable anti-pattern violations.

## Prerequisites (GUARDRAIL)

0. **Change audit:** `python scripts/change_audit.py --staged`
   - Verify that the fix will not exceed change thresholds. If blocked, split fixes across multiple commits.

## Auto-Fixable Rules

- **Rule 2**: Replace `float32`/`float64` price fields with `fixed.Fixed` import suggestion
- **Rule 3**: Convert legacy `.yaml` strategy files to `.gkr.yaml` with `orca convert`
- **Rule 6**: Downgrade `multiplier: 1.0` to `multiplier: 0.25` in GKR configs
- **Rule 10**: Replace `panic(err)` with `return err` in Go files

## Steps

1. Run `python scripts/anti_pattern_scan.py` to identify violations
2. For each auto-fixable violation, apply the fix
3. Re-run anti-pattern scan to verify fixes
4. **Run `python scripts/test_related.py` to verify fixes don't break callers** (GUARDRAIL)
5. If test_related.py fails, revert and report — do NOT proceed
6. Report remaining violations that require manual intervention

## Safety

- Creates backup of original files before modifying
- Only edits files within the current workspace
- Reports diff of all changes made
- **Post-fix test verification required (step 4) — never skip**
