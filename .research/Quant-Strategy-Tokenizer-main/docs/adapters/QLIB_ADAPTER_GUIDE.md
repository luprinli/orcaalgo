# Qlib Adapter Guide

## Command

```bash
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
```

The command writes:

- candidate QST `.gkr.yaml`
- deterministic coverage JSON

It does not import Qlib, run qrun, train models, run inference, execute
backtests, connect to a broker, connect to an exchange, or provide live
execution.

## Supported Proof Case

The supported proof case is:

```text
examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml
```

It records:

- `LGBModel`
- `DatasetH`
- `Alpha158`
- `SignalRecord`
- `PortAnaRecord`
- `TopkDropoutStrategy`
- backtest metadata

Validate the generated candidate:

```bash
python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli canonicalize .local_audit/qlib_lightgbm_alpha158.gkr.yaml --output .local_audit/qlib_lightgbm_alpha158.canonical.json
```

## Partial Cases

These cases intentionally produce partial coverage:

```text
examples/adapters/qlib/workflow_config_custom_model.yaml
examples/adapters/qlib/workflow_config_custom_processor.yaml
```

They record supported workflow structure while flagging custom model or custom
processor components as requiring external Qlib runtime handling.

## Review Rules

- Treat generated GKR as record-layer evidence, not runtime execution.
- Do not add adapter-local refs to the built-in token vocabulary.
- Do not claim lossless conversion.
- Do not weaken reserved, custom-token, broker, exchange, backtest, or live
  execution boundaries.
- Use the coverage JSON to explain what was imported and what remains outside
  QST.
