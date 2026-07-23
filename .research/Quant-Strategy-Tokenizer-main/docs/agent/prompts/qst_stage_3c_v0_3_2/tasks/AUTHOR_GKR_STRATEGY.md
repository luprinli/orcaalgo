# Author Gkr Strategy

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: authoring
foundation: core/00_FOUNDATION.md

## Use When

Use when creating a new editable GKR strategy source.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Preconditions

- classification is `supported` or `partially_supported`
- `token_selection.may_author_gkr` is true
- `repo_context` exists
- at least one current example strategy has been inspected

## Procedure

1. Inspect the closest example strategy under `examples/strategies/<case>/strategy.gkr.yaml`.
2. Inspect dogfood candidate GKR files when the request resembles a dogfood_case.
3. Build a deterministic node plan.
4. Use stable node ids.
5. Use only selected token refs.
6. Use explicit params and explicit input links.
7. Add only required capabilities.
8. Write or patch the `.gkr.yaml` source.
9. Run `qst validate <strategy.gkr.yaml>`.
10. If validation passes, run `qst hash <strategy.gkr.yaml>`.
11. If validation passes, run `qst canonicalize <strategy.gkr.yaml> --output <tmp canonical path>`.
12. If validation fails, route to `tasks/REPAIR_GKR_DIAGNOSTICS.md`.

Candidate GKR files are record-layer evidence. They are not runtime/backtest/profitability
evidence and do not imply broker, exchange, live execution, optimizer, or production
execution support.

## Output

Return a node plan and final authoring evidence:

```yaml
node_plan:
  - node_id:
    token_ref:
    purpose:
    params:
    inputs:
    outputs:
    evidence:
```

```yaml
gkr_authoring:
  file:
  node_plan:
  capabilities:
  validation:
    command:
    result:
    diagnostics:
  hash:
    graph_hash:
    param_hash:
    instance_hash:
  canonical:
    output:
  repair_needed:
  limitations:
  coverage_evidence:
    dogfood_case:
    coverage_row:
    record_layer_only:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- Do not author a strategy when token selection contains reserved blockers or unresolved
  custom token approval requirements.
