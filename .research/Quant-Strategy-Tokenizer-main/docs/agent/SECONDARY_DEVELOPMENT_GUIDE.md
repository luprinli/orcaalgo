# QST Secondary Development Guide

This guide describes safe development after final closure.

## Development Principles

- Prefer small, scoped changes.
- Inspect current code and tests before editing.
- Preserve `qst-ir/0.4` and `qst-canonical/0.4` unless explicitly creating an incompatible research fork.
- Preserve token maturity, execution-support, reserved, non-goal, and custom-runtime boundaries.
- Update tests and docs together.

## Common Change Types

Safe archival changes:

- documentation clarification
- conformance test updates
- coverage report correction
- prompt pack repair
- adapter boundary clarification
- final acceptance update

High-risk changes requiring explicit reopening:

- token vocabulary expansion
- new TypeSpec
- runtime execution surface
- broker or exchange integration
- full backtest engine
- parser or authoring DSL
- adapter importer implementation

## Required Reconnaissance

Before changing behavior, inspect:

```text
README.md
docs/FINAL_SCOPE.md
docs/NON_GOALS.md
docs/token_family_registry.md
docs/token_coverage.md
docs/reports/strategy_coverage_matrix.yaml
qst/
tests/
```

## Required Gates

Run focused tests for the touched area plus:

```bash
python -m ruff check .
python -m mypy qst
python -m qst.cli vocabulary --check
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
```

Run full tests before any release or public push:

```bash
python -m pytest tests -q
python -m pytest --cov=qst --cov-fail-under=85 -q
```
