# QST Token Registration Guide

This guide explains when and how to register a new QST token.

## Token Definition

A token is a stable, typed, versioned record-layer semantic unit.

A token is not:

- an arbitrary user function
- a one-off strategy idea
- a broker operation
- a live execution action
- an opaque Python callback

## Add a Token Only If

A new token is allowed only if:

```text
1. The semantic unit is stable.
2. It is reusable across multiple strategy patterns.
3. It has clear input ports.
4. It has clear output ports.
5. It has clear parameter schema.
6. It has explicit temporal semantics.
7. It can be tested by behavior contracts.
8. It improves direct built-in record coverage.
9. It does not weaken reserved/non-goal boundaries.
```

## Do Not Add a Token If

Do not add a token if:

```text
1. It is a one-off user method.
2. It exists only to improve coverage percentage.
3. It hides unsupported Python logic.
4. It requires live broker or exchange execution.
5. It cannot be tested.
6. It has unclear temporal behavior.
7. It should be a recipe instead.
8. It should be a custom token instead.
```

## Token vs Recipe vs Custom Token

Use this decision tree:

```text
Can existing tokens express it?
  yes -> recipe

Does it introduce stable reusable semantics?
  yes -> built-in token candidate

Is it user-specific or proprietary logic?
  yes -> custom token

Does it require broker/live execution/HFT runtime?
  yes -> reserved or non_goal

Can it not be audited?
  yes -> opaque / unsupported
```

## Required Token Artifacts

A valid token addition must include:

```text
TokenSpec
TokenSurfaceSpec
TokenPack entry
input ports
output ports
params schema
behavior_version
maturity
execution_support
profile support
behavior contracts
unit/conformance tests
docs/token_family_registry.md update
docs/token_coverage.md update
coverage matrix update when relevant
```

## Required Commands

After adding a token:

```bash
python -m ruff check .
python -m mypy qst
python -m pytest tests/token_conformance -q
python -m qst.cli vocabulary --check
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
```
