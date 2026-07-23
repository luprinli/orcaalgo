# Repo Context Schema

prompt_system_version: qst-stage-3c-v0.3.2.3
schema_type: prompt_contract

## Purpose

Define the evidence map for current repository context.

## Required Fields

- task, files_read, commands_run, facts, conflicts, open_questions

## Validation Rules

- facts must cite files or commands and conflicts must be preserved.
- Missing fields must be reported explicitly instead of inferred.
- Output should be deterministic and compact enough for review.

## Output

Return a mapping or report that follows the required field list and preserves unresolved
questions for the next agent.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
