# Generate Handoff

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: handoff
foundation: core/00_FOUNDATION.md

## Use When

Use before stopping with unfinished work or after a multi-step repair.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Restate the task in QST terms and identify the active profile or artifact.
2. Read the current repository evidence before making token, schema, or runtime claims.
3. Apply the relevant token surface, profile gate, security, and hash boundaries.
4. Make the smallest safe change or produce the requested review.
5. Run focused validation and report any skipped gates.

## Output

Return a concise handoff another agent can execute.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
