# Agent Playbook

## Purpose

This playbook describes how an agent should work in the closed QST repository.
Use it with the active prompt pack, not instead of it.

## Standard Workflow

1. Inspect `git status --short` and preserve existing work.
2. Read the relevant final handoff, coverage, token, prompt, or adapter docs.
3. Make the smallest scoped change.
4. Run focused tests first.
5. Run repo gates when the change affects shared behavior.
6. Report exact commands, results, and residual limitations.

## Common Commands

```bash
python -m qst.cli vocabulary --check
python -m qst.cli validate examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli hash examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli canonicalize examples/strategies/kdj_cross_basic.gkr.yaml --output .local_audit/final_kdj.canonical.json
python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
```

## Qlib Adapter Handling

Use the Qlib adapter only for partial record import evidence. Generated GKR is
validatable and hashable QST record material. It is not a Qlib runtime, not a
qrun wrapper, not model training, not inference, not a backtester, not broker
or exchange integration, and not live execution.

Custom Qlib model or processor workflow examples should remain partial and must
surface coverage warnings instead of silently claiming support.

## Stop Conditions

Stop and escalate if the requested task requires:

- modifying `qst-ir/0.4` without explicit scope.
- changing canonical or hash algorithms.
- weakening reserved or non-goal boundaries.
- treating adapter-local refs as built-in accepted tokens.
- executing Qlib, external strategy code, broker calls, or exchange calls.
