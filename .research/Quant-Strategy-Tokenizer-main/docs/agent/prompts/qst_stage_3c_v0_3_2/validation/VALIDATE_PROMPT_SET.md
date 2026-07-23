# Validate Prompt Set

prompt_system_version: qst-stage-3c-v0.3.2.3
validation_type: prompt_pack

## Purpose

Validate that the active prompt pack is present, parseable, readable, content-complete,
and aligned with QST boundaries.

## Required Checks

- Required file existence and version consistency.
- Markdown readability, section completeness, and minimum content density.
- Stale current-state, stale suffix, and hardcoded hash-truth bans.
- Cross-reference integrity and load-profile reference integrity.
- Golden YAML parsing and complete-task schema checks.
- Reserved-design and custom-token separation rules.

## Execution

Run `tools/validate_prompt_set.py` on this prompt root before committing prompt changes.
For remote evidence, run `tools/verify_prompt_remote_artifacts.py` after push against a
commit-specific raw artifact base.

## Output

The validator emits deterministic JSON. A pass means the prompt pack is structurally and
content-wise acceptable; it does not prove QST runtime or trading behavior.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
