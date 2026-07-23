# QST Custom Token Guide

Custom tokens cover user-specific or proprietary logic that should not become a
built-in token.

## Boundary

Custom-token verification may inspect metadata, hashes, and declared ports.

Verification must not:

- import custom Python
- execute custom Python
- approve trust
- create grants
- hide unsupported behavior

Execution requires:

- explicit user request
- integrity verification
- local approval
- execution grant
- current UTC time
- run id
- user-approved inputs
- output validation

## When to Use a Custom Token

Use a custom token when:

- logic is proprietary or strategy-specific
- a stable reusable token contract is not justified
- the input/output ports can still be declared
- the user accepts local execution responsibility

Do not use a custom token to bypass reserved or non-goal boundaries.

## Reference Example

The current example is:

```text
examples/custom_token/kalman/
```

Useful commands:

```bash
python -m qst.cli token verify --help
python -m qst.cli token approve --help
python -m qst.cli token execute --help
```

## Governance Evidence

Coverage Frontier custom-token route evidence lives in:

```text
docs/reports/custom_token_governance_review.md
tests/coverage_cases/custom_token_governance/custom_token_routes.yaml
```

Custom-token routes are discounted in frontier reporting and capped by the
published route-share threshold.
