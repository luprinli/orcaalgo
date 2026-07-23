# /regression-guard — Guardian Regression Smoke Tests

Guardian test suite covering 20 critical code paths. Failure blocks all merges.

## Prerequisites (GUARDRAIL)

0. Verify all 20 critical paths are actually covered by tests. If any path is untested, report as WARN.

## Steps

1. `pytest tests/guardian/ -v --tb=short` — Python guardian tests
2. `go test -tags=guardian ./tests/guardian/ -v` — Go guardian tests
3. Report with file:line references for each of the 20 critical paths
4. Verify coverage: confirm that all 20 paths map to at least one test case
5. If any path is uncovered, log WARN with the path name

## Exit Code

- 0: All 20 critical paths tested and passing
- 1: One or more guardian tests failed
