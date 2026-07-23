# 01 Repo Context Protocol

prompt_system_version: qst-stage-3c-v0.3.2.3
layer: core

## Purpose

Tie current-state claims to files, tests, schemas, and command evidence. This prompt is loaded before task-specific instructions and sets boundaries that
remain active for the entire interaction.

## Required Reconnaissance

Run or inspect the smallest useful set of repository facts before selecting tokens or
authoring a strategy:

```bash
git status --short
git rev-parse --abbrev-ref HEAD
git rev-parse HEAD
cat pyproject.toml
python -m qst.cli --help
python -m qst.cli vocabulary --check
qst vocabulary --check
find docs -maxdepth 3 -type f | sort
find examples/strategies -maxdepth 3 -type f | sort
find tests/reference -maxdepth 4 -type f | sort
python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml
python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check
```

For strategy authoring tasks, also inspect one current example and run smoke commands
when the local environment permits:

```bash
qst validate examples/strategies/01_ema_cross/strategy.gkr.yaml
qst hash examples/strategies/01_ema_cross/strategy.gkr.yaml
qst canonicalize examples/strategies/01_ema_cross/strategy.gkr.yaml --output /tmp/qst_agent_probe.canonical.json
```

For strategy classification, authoring, repair, and review tasks, also inspect Coverage
Frontier evidence when present:

- `docs/reports/strategy_coverage_matrix.yaml`
- `docs/reports/strategy_coverage_report.md`
- `docs/reports/dogfood_target_set.md`
- `docs/reports/original_failure_strategy_dogfood.md`
- `tests/coverage_cases/dogfood/`

Treat coverage rows and dogfood_case records as routing evidence. They do not change QST
semantics and do not prove runtime, backtest, broker, exchange, profitability, or
production execution coverage.
Use `report_strategy_coverage.py --check` as the coverage report gate shorthand when
summarizing this evidence.

If a command cannot be run, record it explicitly:

```yaml
commands_not_run:
  - command:
    reason:
```

## Operating Rules

- QST is a typed and canonical strategy IR system, not a trading bot or broker adapter.
- Repository evidence outranks memory, old reviews, generated summaries, and unstated assumptions.
- Accepted token governance does not automatically imply runtime execution support.
- Reserved design families may be explained, but they must not be turned into executable plans.
- Custom token work must preserve verify, approve, grant, execute, and output validation boundaries.

## Required Output

Return a `repo_context` record before token selection or GKR authoring:

```yaml
repo_context:
  git:
    branch:
    head:
    dirty:
  package:
    import_package:
    cli:
    distribution:
  strategy_format:
    editable_source_suffix:
    package_suffix:
    ir_schema:
    canonical_schema:
  active_docs:
    read:
    missing:
  examples:
    strategy_index:
    sample_strategy:
  token_surface:
    vocabulary_command:
    status:
    diagnostics:
  coverage_frontier:
    strategy_coverage_matrix:
    strategy_coverage_report:
    dogfood_case:
    external_benchmark:
    custom_token_route_share:
    false_supported_rate:
    kernel_gap_count:
  validation_smoke:
    command:
    result:
  hash_smoke:
    command:
    graph_hash:
    param_hash:
    instance_hash:
  limitations:
  commands_not_run:
```

State the selected task, files or commands used as evidence, the concrete decision made,
and any residual risk. If evidence is missing, say what is missing instead of filling the
gap from memory.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- Do not proceed to token selection or GKR authoring unless `repo_context.package`,
  `repo_context.strategy_format`, and `repo_context.token_surface` are known or explicitly
  marked unavailable.
