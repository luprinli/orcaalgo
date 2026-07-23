# Usage Guide

## Install

```bash
pip install -e ".[dev]"
```

## Vocabulary And Strategy Records

```bash
python -m qst.cli vocabulary --check
python -m qst.cli validate examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli hash examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli canonicalize examples/strategies/kdj_cross_basic.gkr.yaml --output .local_audit/final_kdj.canonical.json
```

## Coverage Frontier

```bash
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --json
```

## Prompt Pack Validation

```bash
python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2
python tools/verify_prompt_remote_artifacts.py docs/agent/prompts/qst_stage_3c_v0_3_2
```

## Qlib Partial Adapter Proof

```bash
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli canonicalize .local_audit/qlib_lightgbm_alpha158.gkr.yaml --output .local_audit/qlib_lightgbm_alpha158.canonical.json
```

The Qlib adapter is partial and not lossless. It reads YAML and writes record
evidence. It does not import Qlib, run qrun, train models, run inference,
execute backtests, connect to a broker, connect to an exchange, or provide live
execution.

## Custom Tokens

```bash
python -m qst.cli token verify --help
python -m qst.cli token approve --help
python -m qst.cli token execute --help
```

Custom token execution remains explicit and approval-bound. Do not treat
verification as approval, and do not treat approval as an execution grant.
