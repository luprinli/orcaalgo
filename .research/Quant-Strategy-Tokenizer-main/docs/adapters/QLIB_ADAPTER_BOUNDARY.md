# Qlib Adapter Boundary

## Status

The Qlib adapter is a partial workflow adapter proof.

It imports selected Qlib workflow YAML structure into QST record-layer candidate
GKR and coverage evidence. It is not a Qlib runtime replacement and not a
lossless converter.

## Included

- `qst/adapters/qlib/` Python package.
- `qst adapter qlib import` CLI group and import command.
- YAML loader that reads workflow config without importing Qlib or executing
  Python.
- Coverage extraction for model, dataset, records, `TopkDropoutStrategy`, and
  backtest metadata.
- Candidate `.gkr.yaml` generation using adapter-local token refs in the
  `adapter` namespace.
- Example workflow configs under `examples/adapters/qlib/`.
- Tests under `tests/adapters/qlib/`.

## Not Included

The Qlib adapter must not be described as any of these:

- Qlib runtime replacement.
- qrun execution.
- model training.
- inference execution.
- backtest execution.
- broker integration.
- exchange integration.
- live execution.
- lossless Qlib conversion.
- optimizer or portfolio execution engine.

## Adapter-Local Token Refs

Generated candidate GKR uses adapter-local record token refs such as:

```text
adapter.data.qlib_dataset_record/v1/bv1
adapter.model.forecast_model_record/v1/bv1
adapter.record.signal_record/v1/bv1
```

These refs are not added to `builtin_token_packs()` and are not accepted core
token vocabulary. They are candidate record evidence for review, validation,
hashing, and canonicalization.

## Classification

The importer emits deterministic coverage JSON:

- `supported`: supported record-layer workflow structure.
- `partially_supported`: record-layer structure plus custom or opaque Qlib
  component.
- `custom_token_required`: custom behavior with no supported record structure.
- `reserved`: reserved design boundary.
- `non_goal`: out-of-scope runtime or execution requirement.

`supported` means record-layer import support only. It does not mean the Qlib
workflow can be run by QST.
