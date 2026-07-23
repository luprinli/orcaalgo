# Final Handoff

## Entry Point

QST is archived as an agent-ready research prototype.

Start with:

- [Final Scope](FINAL_SCOPE.md)
- [Final Acceptance](FINAL_ACCEPTANCE.md)
- [Final Report](FINAL_REPORT.md)
- [Agent Takeover Prompt](agent/AGENT_TAKEOVER_PROMPT.md)
- [Agent Playbook](agent/AGENT_PLAYBOOK.md)
- [Usage Guide](agent/USAGE_GUIDE.md)
- [Qlib Adapter Boundary](adapters/QLIB_ADAPTER_BOUNDARY.md)
- [Qlib Adapter Guide](adapters/QLIB_ADAPTER_GUIDE.md)

## Current Truth

- Python package: `qst`.
- Strategy source: `.gkr.yaml`.
- Active IR: `qst-ir/0.4`.
- Active canonical schema: `qst-canonical/0.4`.
- Active prompt pack: `docs/agent/prompts/qst_stage_3c_v0_3_2/`.
- Coverage Frontier report: `docs/reports/strategy_coverage_report.md`.
- Qlib adapter proof: `qst adapter qlib import`.

## Qlib Adapter Usage

```bash
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli canonicalize .local_audit/qlib_lightgbm_alpha158.gkr.yaml --output .local_audit/qlib_lightgbm_alpha158.canonical.json
```

Treat generated artifacts as record-layer evidence only. They do not prove
runtime execution, backtesting, broker integration, exchange integration, live
execution, or profitability.

## Handoff Rules

- Preserve QST boundaries before extending functionality.
- Do not add token coverage merely to improve a percentage.
- Do not weaken reserved, non-goal, or custom runtime boundaries.
- Keep adapter-local token refs outside built-in vocabulary unless a future
  explicit token-surface process accepts them.
- Run the final acceptance gates before claiming closure.
