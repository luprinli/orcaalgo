# Classify Strategy Intent

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: authoring
foundation: core/00_FOUNDATION.md

## Use When

Use before selecting tokens for a strategy request.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Restate the strategy intent in one sentence.
2. Extract required components:
   - data
   - indicators
   - transforms
   - decision logic
   - state/gates
   - panel/universe
   - risk/weight
   - custom logic
   - external/runtime requirements
3. Check external/runtime requirements before token selection.
4. Check current vocabulary, examples, strategy_coverage_matrix, external_benchmark rows,
   dogfood_case evidence, and coverage reports through repository evidence.
5. Record false_supported_rate risk when a request resembles a supported row but lacks
   mechanical, semantic, or boundary evidence.
6. Classify using exactly one allowed classification.
7. Produce a strategy intent record.

## Classification Rules

- `supported`: current vocabulary and schema can express the strategy, the target profile
  allows selected token families, and no unsupported runtime or type requirement appears.
- `partially_supported`: the core strategy can be expressed, but one or more non-essential
  elements must be omitted, reported, or routed without silent weakening.
- `custom_token_required`: missing logic is representable as a custom token without an
  unsupported base type or reserved runtime; custom token approval and execution are not
  implied.
- `reserved`: the request requires a reserved design surface such as EventStream, broad
  stream runtime, unsupported type boundary, HFT/event-time replay, or distribution type.
- `non_goal`: the request requires broker, exchange, live execution, order routing,
  custody, production runtime, or a broad backtest engine.

## Allowed Classifications

- supported
- partially_supported
- reserved
- custom_token_required
- non_goal

## Output

Return a strategy intent record using `schemas/STRATEGY_INTENT_SCHEMA.md`:

```yaml
strategy_intent:
  intent_summary:
  target_profile:
  classification:
  reason:
  required_data:
  signal_logic:
  decision_logic:
  state_requirements:
  panel_requirements:
  risk_weight_requirements:
  external_requirements:
  token_families:
  missing_tokens:
  missing_types:
  blockers:
  coverage_evidence:
    coverage_row:
    external_benchmark:
    dogfood_case:
    false_supported_rate:
    kernel_gap:
  next_task:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- Do not fake reserved EventStream, HFT replay, broker, exchange, or live execution
  requirements with ordinary time-series tokens.
