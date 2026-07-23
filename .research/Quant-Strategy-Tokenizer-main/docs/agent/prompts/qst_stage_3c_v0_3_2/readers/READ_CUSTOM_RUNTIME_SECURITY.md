# Custom Runtime Security

prompt_system_version: qst-stage-3c-v0.3.2.3
reader_type: repository_reader

## Purpose

Use this reader when the task depends on current repository evidence about custom token verify, approval, grant, execute, and audit boundaries.
The reader gathers facts; it does not decide or edit by itself.

## Read

- Active code, docs, tests, schemas, examples, or CI files that define custom token verify, approval, grant, execute, and audit boundaries.
- Adjacent tests that prove the behavior or boundary.
- Reference artifacts only when they are part of the current product surface.

## Extract

- Stable facts that can be tied to file paths or command output.
- Contradictions between implementation, tests, and docs.
- Missing tests, stale claims, or unsupported capability wording.

## Report

Return a concise module report with inspected files, facts learned, and remaining risk.
If stale information appears, route the task through `tasks/REPAIR_STALE_INFORMATION.md`.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
