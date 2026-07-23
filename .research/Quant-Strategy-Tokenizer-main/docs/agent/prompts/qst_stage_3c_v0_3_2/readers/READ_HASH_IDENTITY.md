# Hash Identity

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about hash classes, hash material, and drift sentinels.
The reader gathers facts; it does not decide or edit by itself.

## Read

Inspect active hash code, CLI hash behavior, reference strategy hash tests, docs, and
`tests/reference` artifacts.

Look for these hash identities:

- graph_hash
- param_hash
- instance_hash
- behavior_hash
- token_hash
- TokenSpec and TokenPack hash material
- excluded material
- sentinel files and tests/reference evidence

## Extract

```yaml
hash_identity:
  classes:
    graph_hash:
    param_hash:
    instance_hash:
    behavior_hash:
    token_hash:
  hash_material:
  excluded_material:
  sentinel_paths:
  drift_policy:
  command_evidence:
```

Also extract contradictions between implementation, tests, docs, examples, and hash
sentinels.

## Report

Return a concise module report with inspected files, hash facts, drift risks, and any
missing sentinel evidence.
If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
