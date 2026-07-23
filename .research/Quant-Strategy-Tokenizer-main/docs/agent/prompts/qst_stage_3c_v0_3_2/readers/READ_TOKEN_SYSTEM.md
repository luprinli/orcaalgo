# Token System

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about TokenSpec, TokenSurfaceSpec, TokenPack, vocabulary, and token contracts.
The reader gathers facts; it does not decide or edit by itself.

## Read

Inspect these first:

- `docs/token_family_registry.md`
- `docs/token_coverage.md`
- `docs/reference.md`
- token modules under `qst/`
- tests matching `*token*`
- examples using token refs

Run when available:

```bash
python -m qst.cli vocabulary --check
qst vocabulary --check
```

Use current `TokenSpec`, `TokenSurfaceSpec`, `TokenPack`, vocabulary, and token contract
code as source evidence. Adjacent tests prove behavior or boundary; reference artifacts
are evidence only when they are part of the current product surface.

## Extract

```yaml
token_system:
  token_ref_format:
  families:
  maturities:
  execution_support_values:
  reserved_design_families:
  custom_runtime_boundaries:
  vocabulary_command:
  diagnostics:
  source_files:
```

Also extract stable facts tied to file paths or command output, contradictions between
implementation/tests/docs, missing tests, stale claims, and unsupported capability
wording.

## Report

Return both a concise module report and a `vocabulary_snapshot`:

```yaml
vocabulary_snapshot:
  tokens:
    - token_ref:
      family:
      maturity:
      execution_support:
      profile_status:
      source_evidence:
  missing_or_ambiguous:
  rejected_assumptions:
```

If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
