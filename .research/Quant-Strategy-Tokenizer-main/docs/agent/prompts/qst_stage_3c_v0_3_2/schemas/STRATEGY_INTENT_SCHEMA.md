# Strategy Intent Schema

prompt_system_version: qst-stage-3c-v0.3.2.3
schema_type: prompt_contract

## Purpose

Define the record produced by strategy intent classification.

## Required Fields

```yaml
strategy_intent:
  intent_summary:
  target_profile:
  classification:
  reason:
  universe:
    kind:
    assets:
  required_data:
    - name:
      type:
      timeframe:
  signal_logic:
  decision_logic:
  state_requirements:
  panel_requirements:
  risk_weight_requirements:
  external_requirements:
    broker:
    exchange:
    live_execution:
    order_routing:
    backtest_engine:
    hft_runtime:
    event_stream:
    distribution_type:
  token_families:
  missing_tokens:
  missing_types:
  blockers:
  next_task:
```

## Validation Rules

- classification must be supported, partially_supported, reserved, custom_token_required, or non_goal.
- Missing fields must be reported explicitly instead of inferred.
- Output should be deterministic and compact enough for review.
- external requirements must distinguish broker, exchange, live execution, EventStream,
  HFT runtime, and distribution type blockers.

## Output

Return a mapping or report that follows the required field list and preserves unresolved
questions for the next agent.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
