# /test-related — Run Tests Only for Changed Files

Detects changed files vs `origin/main` and runs only relevant tests.

## Steps

1. `python scripts/test_related.py --base origin/main`
2. Report with coverage delta

## Exit Code

- 0: All relevant tests PASS
- 1: One or more tests FAILED
- 2: UNTESTED — Changed files map to zero tests (blocking — add tests before merge)

## Language Support

- `--language python` — Python-only test mapping
- `--language go` — Go-only test mapping
- Omitting `--language` runs all detected languages

## Guardrail

If exit code 2 (zero tests for changed files), do NOT bypass. Add test coverage before proceeding.
