# ADR: Token-First Surface After Core

## Status

Accepted for Stage 3A.

## Decision

QST exposes token governance as first-class TokenSpec metadata through
`surface: TokenSurfaceSpec`. Family, category, layer, maturity, execution support,
contracts, capability flags, and agent-facing notes are hash-bearing TokenSpec
material.

`state` remains token-specific implementation/state metadata. It is not the home
for product taxonomy or maturity policy.

## Consequences

- TokenSpec and TokenPack hashes change when surface or contract metadata changes.
- Strategy graph, parameter, and instance hashes do not change solely because a
  TokenSpec surface contract changes.
- `accepted` is not the same as executable. Execution is governed by
  `execution_support`.
- `event.*` and `distribution.*` may appear in vocabulary as `reserved_design`,
  but strategy usage is rejected until their type/runtime layers exist.
- Optimizer tokens remain experimental unless they provide solver determinism
  evidence.
