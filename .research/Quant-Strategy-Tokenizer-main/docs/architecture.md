# Architecture

QST is organized around Graph Kernel Records: typed strategy records with deterministic canonicalization, hashable semantic surfaces, and reference validation artifacts.

## Layers

- `qst.ir`: strategy IR models, canonicalization, validation, and GKR path helpers.
- `qst.types`: TypeSpec parsing and structured value types.
- `qst.ports`: input/output signatures and temporal rules.
- `qst.hash`: graph, parameter, behavior, signature, token, runtime, audit, and expected-artifact hashes.
- `qst.tokens`: TokenSpec, TokenSurfaceSpec, TokenPack, lock metadata, built-in vocabulary, and registry resolution.
- `qst.validation`: diagnostics, validation results, and deterministic validator ordering.
- `qst.profiles`: research, paper, pretrade, and guarded-production policy shells.
- `qst.numeric`: numeric policy and token evolution metadata.
- `qst.state`: delay, accumulate, edge-detect, FSM, and state reference traces.
- `qst.decision`: decision kinds, monoids, fold policies, aggregators, and migration classification helpers.
- `qst.panel`: panel type-layer metadata, panel reference operators, weight operators, and reference validation runners.
- `qst.custom_runtime`: custom-token integrity, local approval, execution grants, audit records, and output validation.

## Canonical Material

The canonical surface is JSON-compatible and deterministic. Semantic model changes enter the appropriate hash material; trace output and local approval state do not enter strategy identity.

Token surface metadata is canonical TokenSpec material. It changes TokenSpec and
TokenPack hashes, but it does not alter strategy graph or parameter hashes unless
the strategy graph or params themselves change.

## Public Artifacts

- Source strategies use `.gkr.yaml`.
- Packaged Graph Kernel Records reserve `.gkr`.
- JSON schemas live in `docs/schemas/` and keep their internal `$id` and `schema_version` values stable.
- Deterministic reference artifacts live in `tests/reference/`.
- Stage 3B token-surface acceptance and gap-review reports live in `docs/reports/`.
