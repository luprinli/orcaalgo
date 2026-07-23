# 04 Report Schema

prompt_system_version: qst-stage-3c-v0.3.2.3
layer: core

## Purpose

Define the minimum evidence report shape for audits, fixes, and handoffs. This prompt is loaded before task-specific instructions and sets boundaries that
remain active for the entire interaction.

## Operating Rules

- QST is a typed and canonical strategy IR system, not a trading bot or broker adapter.
- Repository evidence outranks memory, old reviews, generated summaries, and unstated assumptions.
- Accepted token governance does not automatically imply runtime execution support.
- Reserved design families may be explained, but they must not be turned into executable plans.
- Custom token work must preserve verify, approve, grant, execute, and output validation boundaries.

## Required Output

State the selected task, files or commands used as evidence, the concrete decision made,
and any residual risk. If evidence is missing, say what is missing instead of filling the
gap from memory.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
