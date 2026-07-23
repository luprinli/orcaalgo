# Reference

## Strategy Files

Editable strategy files use `.gkr.yaml`. CLI strategy inputs reject other source suffixes.

```bash
qst validate examples/strategies/kdj_cross_basic.gkr.yaml
qst hash examples/strategies/kdj_with_ema_filter.gkr.yaml
```

The public Stage 3A examples are indexed in
[`examples/strategies/README.md`](../examples/strategies/README.md). Each public
demo has matching validation diagnostics and graph/param/instance hash
sentinels under `tests/reference/strategies/<case>/`.

## Schemas

Public schema files live in `docs/schemas/`. File names are public product names; internal `$id` and `schema_version` values remain stable.

Important schemas include:

- `ir-0.4.schema.json`
- `type_spec-0.4.schema.json`
- `port_spec-0.4.schema.json`
- `token_spec-0.4.schema.json`
- `token_surface-0.4.schema.json`
- `token_pack-0.4.schema.json`
- `decision-0.4.schema.json`
- `state_policy-0.4.schema.json`
- `panel_representation-0.4.schema.json`
- `approval_record-0.4.schema.json`
- `execution_grant-0.4.schema.json`

## Reference Data

Deterministic reference material lives under `tests/reference/`:

- `temporal/`
- `state/`
- `panel/`
- `custom_token/kalman/`
- `strategies/`

These fixtures and traces are conformance material for the current QST tree, not runtime execution logs.

The Stage 3A demo acceptance gate requires all 12 public examples to validate
through both the Python API and CLI, and to match their reference hash
sentinels. Full trace artifacts are intentionally limited to
`01_ema_cross`, `08_market_neutral_rank`, and
`12_custom_token_kalman_signal`.

## Token Surface

Built-in tokens are exposed through `builtin_token_packs()` and each TokenSpec
contains `surface: TokenSurfaceSpec`. Surface metadata declares family, category,
layer, maturity, execution support, contracts, capability flags, and agent-facing
notes. Surface metadata is TokenSpec hash material.

See [token_family_registry.md](token_family_registry.md) and
[token_coverage.md](token_coverage.md).

Stage 3B acceptance reports live in `docs/reports/`:

- `token_surface_acceptance.md`
- `token_contract_audit.md`
- `token_coverage_review.md`
- `token_gap_report.md`
- `stage_3b_decision.md`

## Numeric Status

Panel and weight helpers define semantic float64 reference behavior. They are deterministic for the provided reference inputs but are not a bit-exact numerical portability claim across every possible runtime.
