# Validation Diagnostics

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about validation diagnostics, profile gates, and stable error reporting.
The reader gathers facts; it does not decide or edit by itself.

## Read

Inspect:

- validation entrypoint files under `qst/`
- validator tests
- diagnostic model tests
- profile gate tests
- `tests/reference/strategies/<case>/diagnostics*`
- CLI validation command behavior

Run when available:

```bash
qst validate examples/strategies/01_ema_cross/strategy.gkr.yaml
qst validate examples/strategies/12_custom_token_kalman_signal/strategy.gkr.yaml
```

## Extract

```yaml
validation_diagnostics:
  entrypoints:
  diagnostic_shape:
  severity_values:
  stable_ordering:
  profile_gate_behavior:
  reserved_design_behavior:
  common_error_classes:
    - schema_error
    - unknown_token
    - profile_blocked_token
    - reserved_design_token
    - port_error
    - type_error
    - temporal_error
    - capability_error
```

Also extract contradictions between implementation, tests, docs, and reference artifacts.

## Report

Return a concise module report with inspected files, diagnostic facts, profile gate facts,
and remaining risk.
If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
