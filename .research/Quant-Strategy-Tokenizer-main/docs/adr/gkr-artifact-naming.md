# ADR: GKR Artifact Naming

Date: 2026-05-16

## Decision

QST uses Graph Kernel Record (GKR) as the public artifact family name.

- Editable strategy source: `.gkr.yaml`
- Packaged Graph Kernel Record: `.gkr`
- Python import package: `qst`
- CLI command: `qst`
- Distribution name: `quant-strategy-tokenizer`
- Internal schema identity: `qst-ir/0.4` and `qst-canonical/0.4`

## Rationale

The public name should describe the artifact as a typed graph record rather than an execution recipe. Keeping the distribution name stable avoids package publication churn while the import package and CLI become concise.

## Consequences

The CLI accepts `.gkr.yaml` as strategy source. The `.gkr` suffix is reserved for packaged records, but this tree does not add a packaged runtime.