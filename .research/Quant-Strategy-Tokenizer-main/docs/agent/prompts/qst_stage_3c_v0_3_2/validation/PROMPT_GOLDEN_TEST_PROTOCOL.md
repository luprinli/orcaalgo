# Prompt Golden Test Protocol

prompt_system_version: qst-stage-3c-v0.3.2.3
validation_type: golden_tasks

## Purpose

Use golden tasks to check that the prompt system routes common, custom-token, and
reserved-design strategy intents consistently.

## Required Checks

- `golden/01_ema_cross.intent.yaml` remains a supported strategy classification.
- `golden/12_custom_token_kalman_signal.intent.yaml` remains custom token required.
- `golden/13_event_stream_intraday.intent.yaml` remains reserved.
- Skeleton tasks may exist, but complete tasks must include intent, expected output,
  forbidden behavior, and acceptance criteria.

## Execution

Parse every golden YAML file, validate the expected classification, then run representative
task prompts manually or through an evaluator when available.

## Output

Report the prompt task, expected classification, actual classification, and any mismatch.
Do not treat a golden task as evidence of broad runtime execution.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
