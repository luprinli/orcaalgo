# QST Recipe Authoring Guide

Recipes are reusable compositions of existing QST tokens. Use a recipe when a
strategy pattern can be expressed without adding a new semantic primitive.

## When to Use a Recipe

Use a recipe for:

- indicator compositions
- signal compositions
- common gate compositions
- standard feature transforms
- strategy templates that reuse accepted tokens

Do not create a new token if a recipe is sufficient.

## Recipe Boundary

A recipe must not:

- hide unsupported Python logic
- weaken profile gates
- execute custom code
- use reserved-design tokens as executable behavior
- imply broker, exchange, or live execution support

## Authoring Checklist

Before adding a recipe, record:

```text
strategy intent
selected token refs
input requirements
output record
params
temporal assumptions
profile support
coverage rows affected
examples or tests
```

## Validation

Use current GKR examples as style evidence:

```text
examples/strategies/
tests/coverage_cases/
tests/reference/
```

Run:

```bash
python -m qst.cli validate <candidate>.gkr.yaml
python -m qst.cli hash <candidate>.gkr.yaml
python -m qst.cli canonicalize <candidate>.gkr.yaml --output .local_audit/<candidate>.canonical.json
```

If the recipe needs a new TypeSpec, EventStream, Distribution, broker feedback,
or execution runtime, classify it as reserved or non-goal instead.
