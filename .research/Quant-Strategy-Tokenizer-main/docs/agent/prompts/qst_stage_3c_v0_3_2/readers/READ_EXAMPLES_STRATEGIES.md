# Examples Strategies

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about public example strategies and reference strategy artifacts.
The reader gathers facts; it does not decide or edit by itself.

## Read

Inspect:

- `examples/strategies/README.md`
- `examples/strategies/<case>/strategy.gkr.yaml`
- `tests/reference/strategies/<case>/`
- related validation diagnostics
- related hash sentinels

Use examples as schema and style evidence, not as proof of profitability or runtime
support.

## Extract

```yaml
examples_strategies:
  cases:
    - id:
      path:
      focus:
      token_families:
      full_trace:
      validation_command:
      hash_command:
      reference_path:
  authoring_patterns:
  schema_patterns:
  node_id_patterns:
  capability_patterns:
```

Also extract contradictions between examples, tests, docs, and current CLI behavior.

## Report

Return a concise module report with example cases inspected, reusable authoring patterns,
and any reference artifact gaps.
If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
