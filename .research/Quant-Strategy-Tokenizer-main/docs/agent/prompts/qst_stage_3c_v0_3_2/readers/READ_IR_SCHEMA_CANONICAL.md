# Ir Schema Canonical

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about IR schema identity and canonical representation.
The reader gathers facts; it does not decide or edit by itself.

## Read

Inspect active schema docs, schema files, canonicalization code, validation code, examples,
and tests that prove current `qst-ir/0.4` and `qst-canonical/0.4` behavior.

Pay special attention to:

- schema_version
- canonical_version
- node shape
- token_ref shape
- input link shape
- metadata included/excluded from canonical JSON
- deterministic ordering rules

## Extract

```yaml
ir_canonical:
  ir_schema:
  canonical_schema:
  node_shape:
  token_ref_shape:
  input_link_shape:
  canonical_json_rules:
  included_material:
  excluded_material:
  validation_entrypoints:
```

Also extract contradictions between implementation, tests, docs, and examples.

## Report

Return a concise module report with current schema/canonical facts, inspected files, and
remaining risk.
If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
