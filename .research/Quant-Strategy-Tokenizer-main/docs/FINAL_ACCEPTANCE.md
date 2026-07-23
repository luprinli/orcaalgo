# Final Acceptance

## Scope

Final acceptance covers the archived agent-ready research prototype plus the
Qlib partial workflow adapter proof.

## Accepted Claims

- Python package: `qst`.
- Editable strategy source suffix: `.gkr.yaml`.
- Active IR and canonical schema: `qst-ir/0.4` and `qst-canonical/0.4`.
- Available CLI includes `vocabulary`, `validate`, `hash`, `canonicalize`,
  `token verify`, `token approve`, `token execute`, and
  `adapter qlib import`.
- Qlib adapter proof imports workflow YAML into candidate GKR and coverage JSON.
- Qlib adapter proof is partial and not lossless.

## Rejected Claims

QST does not provide:

- Qlib runtime replacement.
- qrun execution.
- model training.
- inference execution.
- backtest execution.
- broker or exchange integration.
- live execution.
- lossless Qlib conversion.
- full broad strategy runtime.

## Local Evidence

The following commands are expected final gates for this closure:

```bash
python -m pytest tests/adapters/qlib -q
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli canonicalize .local_audit/qlib_lightgbm_alpha158.gkr.yaml --output .local_audit/qlib_lightgbm_alpha158.canonical.json
python -m compileall qst
python -m ruff check .
python -m mypy qst
python -m qst.lint.stateless qst
python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2
python tools/verify_prompt_remote_artifacts.py docs/agent/prompts/qst_stage_3c_v0_3_2
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
python -m qst.cli vocabulary --check
python -m qst.cli validate examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli hash examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli canonicalize examples/strategies/kdj_cross_basic.gkr.yaml --output .local_audit/final_kdj.canonical.json
python -m pytest tests/docs -q
python -m pytest tests -q
python -m pytest --cov=qst --cov-fail-under=85 -q
git diff --check
```

The final commit SHA is intentionally not recorded in this file because the
file is part of that commit. The final response should report the commit SHA,
push result, and tag result after the gates pass.

## Latest Local Run Summary

Recorded on the final closure worktree before commit:

- `python -m pytest tests/adapters/qlib -q`: pass, 8 tests.
- `python -m qst.cli adapter qlib import ...workflow_config_lightgbm_alpha158.yaml`: pass, coverage classification `supported`.
- `python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml`: pass, no diagnostics.
- `python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml`: pass.
  - graph hash: `sha256:c9aea99f9e84170842052c35c1b0e8e18f4c50a4f9981bd47d95929cba2b8be0`
  - param hash: `sha256:afdbb880e2f0b445f274be595f80f825b9d07c61aa44c30fc1256307fcb76867`
  - instance hash: `sha256:74bcc99a24370b99732b134f2f75e96e968ed32c2636341fe8698e976645150a`
- `python -m compileall qst`: pass.
- `python -m ruff check .`: pass.
- `python -m mypy qst`: pass.
- `python -m qst.lint.stateless qst`: pass.
- prompt validation and prompt artifact verification: pass.
- coverage matrix validation and coverage report check: pass.
- `python -m qst.cli vocabulary --check`: pass, 179 tokens across 6 packs.
- KDJ validate/hash/canonicalize smoke: pass.
- `python -m pytest tests/docs -q`: pass, 7 tests.
- `python -m pytest tests -q`: pass, 526 tests.
- `python -m pytest --cov=qst --cov-fail-under=85 -q`: pass, 89.08 percent total coverage.
