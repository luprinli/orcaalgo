# Changelog

## Stage 3A Token Surface Completion

- Added TokenSurfaceSpec metadata and token contract fields to TokenSpec.
- Added deterministic built-in TokenPack vocabulary entrypoint and token conformance gates.
- Added public token-surface demo strategies with validation artifacts and hash sentinels.
- Documented token family maturity, execution support, and hash impact boundaries.
- No broad runtime, broker, backtester, optimizer engine, IR, or canonical hash algorithm change was introduced.

## Public Product Tree Reset

- Renamed the public Python package to `qst` while keeping the distribution name `quant-strategy-tokenizer`.
- Standardized editable strategy files on the `.gkr.yaml` suffix and reserved `.gkr` for packaged Graph Kernel Records.
- Moved public examples, custom-token material, schemas, and deterministic reference fixtures into the product tree layout.
- Rewrote active documentation around the current product boundary.
- No IR, canonical, hash, token, panel, state, decision, or custom-runtime semantics were intentionally changed.
